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
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
)

func NewProcessVM(opts *vmOptions) VM {
	return &ProcessVM{
		path: opts.jsonnetBinaryPath,
		args: opts.args,
		ctx:  opts.ctx,
	}
}

func (p *ProcessVM) EvaluateAnonymousSnippet(filename string, snippet string) (string, error) {
	// We retry the process creation, because it sometimes times out.
	const processVMTimeout = 1 * time.Second
	return backoff.RetryWithData(func() (string, error) {
		ctx, cancel := context.WithTimeout(p.ctx, processVMTimeout)
		defer cancel()

		var (
			stdin          bytes.Buffer
			stdout, stderr strings.Builder
		)
		p.params.Filename = filename
		p.params.Snippet = snippet

		if err := p.params.EncodeTo(&stdin); err != nil {
			return "", backoff.Permanent(errors.WithStack(err))
		}

		cmd := exec.CommandContext(ctx, p.path, p.args...) //nolint:gosec
		cmd.Stdin = &stdin
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		cmd.Env = []string{"GOMAXPROCS=1"}

		err := cmd.Run()
		if stderr.Len() > 0 {
			// If the process wrote to stderr, this means it started and we won't retry.
			return "", backoff.Permanent(fmt.Errorf("jsonnetsecure: unexpected output on stderr: %q", stderr.String()))
		}
		if err != nil {
			return "", fmt.Errorf("jsonnetsecure: %w (stdout=%q stderr=%q)", err, stdout.String(), stderr.String())
		}

		return stdout.String(), nil
	}, backoff.WithContext(backoff.NewExponentialBackOff(), p.ctx))
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
