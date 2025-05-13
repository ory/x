// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package jsonnetsecure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/ory/x/otelx"
)

const (
	KiB                uint64 = 1024
	jsonnetOutputLimit uint64 = 512 * KiB
	jsonnetErrLimit    uint64 = 1 * KiB
)

func NewProcessVM(opts *vmOptions) VM {
	return &ProcessVM{
		path: opts.jsonnetBinaryPath,
		args: opts.args,
		ctx:  opts.ctx,
	}
}

// Jsonnet evaluation is run in a subprocess with a timeout.
// Standard output and error are limited in size and are truncated to this limit
// if too big (but this is not an error condition since that would be too complex to detect and not worth the effort).
func (p *ProcessVM) EvaluateAnonymousSnippet(filename string, snippet string) (_ string, err error) {
	tracer := trace.SpanFromContext(p.ctx).TracerProvider().Tracer("")
	ctx, span := tracer.Start(p.ctx, "jsonnetsecure.ProcessVM.EvaluateAnonymousSnippet", trace.WithAttributes(attribute.String("filename", filename)))
	defer otelx.End(span, &err)

	// We retry the process creation, because it sometimes times out.
	const processVMTimeout = 1 * time.Second
	return backoff.RetryWithData(func() (_ string, err error) {
		ctx, span := tracer.Start(ctx, "jsonnetsecure.ProcessVM.EvaluateAnonymousSnippet.run")
		defer otelx.End(span, &err)

		ctx, cancel := context.WithTimeout(ctx, processVMTimeout)
		defer cancel()

		var (
			stdin bytes.Buffer
		)

		p.params.Filename = filename
		p.params.Snippet = snippet

		if err := p.params.EncodeTo(&stdin); err != nil {
			return "", backoff.Permanent(errors.WithStack(err))
		}

		cmd := exec.CommandContext(ctx, p.path, p.args...) //nolint:gosec
		cmd.Stdin = &stdin
		cmd.Env = []string{"GOMAXPROCS=1"}

		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			return "", backoff.Permanent(errors.WithStack(err))
		}
		defer stdoutPipe.Close()

		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			return "", backoff.Permanent(errors.WithStack(err))
		}
		defer stderrPipe.Close()

		stdoutReader := io.LimitReader(stdoutPipe, int64(jsonnetOutputLimit))
		stderrReader := io.LimitReader(stderrPipe, int64(jsonnetErrLimit))

		if err := cmd.Start(); err != nil {
			return "", backoff.Permanent(fmt.Errorf("jsonnetsecure: failed to start subprocess: %w", err))
		}

		stdoutOutput, err := io.ReadAll(stdoutReader)
		if err != nil {
			return "", backoff.Permanent(fmt.Errorf("jsonnetsecure: failed to read subprocess stdout: %w", err))
		}

		// Reading from stderr is best effort (there might not be anything).
		stderrOutput, _ := io.ReadAll(stderrReader)

		// If there was some stderr output or the stdout has reached the limit,
		// no point in keeping the subprocess running so we kill it.
		// This limits the negative effect of misbehaving jsonnet scripts.
		// NOTE: Depending on what the subprocess does and the OS scheduling, this might kill the subprocess, or have no effect (e.g. the child already terminated).
		if len(stderrOutput) > 0 || len(stdoutOutput) == int(jsonnetOutputLimit) {
			cmd.Cancel()
		}

		err = cmd.Wait()
		if err != nil || len(stderrOutput) > 0 {
			return "", backoff.Permanent(fmt.Errorf("jsonnetsecure: subprocess encountered an error: %w %s", err, string(stderrOutput)))
		}

		return string(stdoutOutput), nil
	}, backoff.WithContext(backoff.NewExponentialBackOff(), ctx))
}

func (p *ProcessVM) ExtCode(key string, val string) {
	p.params.ExtCodes = append(p.params.ExtCodes, kv{key, val})
}

func (p *ProcessVM) ExtVar(key string, val string) {
	p.params.ExtVars = append(p.params.ExtVars, kv{key, val})
}

func (p *ProcessVM) TLACode(key string, val string) {
	p.params.TLACodes = append(p.params.TLACodes, kv{key, val})
}

func (p *ProcessVM) TLAVar(key string, val string) {
	p.params.TLAVars = append(p.params.TLAVars, kv{key, val})
}

func (pp *processParameters) EncodeTo(w io.Writer) error {
	return json.NewEncoder(w).Encode(pp)
}

func (pp *processParameters) DecodeFrom(r io.Reader) error {
	return json.NewDecoder(r).Decode(pp)
}

func (pp *processParameters) Decode(d []byte) error {
	return json.Unmarshal(d, pp)
}
