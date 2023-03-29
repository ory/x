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
	ctx, cancel := context.WithTimeout(p.ctx, 1*time.Second)
	defer cancel()

	var (
		stdin          bytes.Buffer
		stdout, stderr strings.Builder
	)
	p.params.Filename = filename
	p.params.Snippet = snippet

	if err := p.params.EncodeTo(&stdin); err != nil {
		return "", errors.WithStack(err)
	}

	cmd := exec.CommandContext(ctx, p.path, p.args...) //nolint:gosec
	cmd.Stdin = &stdin
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = []string{"GOMAXPROCS=1"}

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("jsonnetsecure: %w (stdout=%q stderr=%q)", err, stdout.String(), stderr.String())
	}
	if stderr.Len() > 0 {
		return "", fmt.Errorf("jsonnetsecure: unexpected output on stderr: %q", stderr.String())
	}

	return stdout.String(), nil
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
