// Copyright © 2024 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package jsonnetsecure

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"math"
	"os/exec"
	"strings"
	"time"

	"github.com/jackc/puddle/v2"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/ory/x/otelx"
)

type (
	processPoolVM struct {
		path   string
		args   []string
		ctx    context.Context
		params processParameters
		pool   *pool
	}
	Pool interface {
		Close()
		private()
	}
	pool struct {
		puddle *puddle.Pool[worker]
	}
	worker struct {
		cmd    *exec.Cmd
		stdin  chan<- []byte
		stdout <-chan string
		stderr <-chan string
	}
	contextKeyType string
)

var (
	ErrProcessPoolClosed = errors.New("jsonnetsecure: process pool closed")

	_ VM   = (*processPoolVM)(nil)
	_ Pool = (*pool)(nil)

	contextValuePath contextKeyType = "argc"
	contextValueArgs contextKeyType = "argv"
)

func NewProcessPool(size int) Pool {
	size = max(1, min(size, math.MaxInt32))
	pud, err := puddle.NewPool(&puddle.Config[worker]{
		MaxSize:     int32(size),
		Constructor: newWorker,
		Destructor:  worker.destroy,
	})
	if err != nil {
		panic(err) // this should never happen, see implementation of puddle.NewPool
	}
	go func() {
		for {
			time.Sleep(10 * time.Second)
			for _, proc := range pud.AcquireAllIdle() {
				if proc.Value().cmd.ProcessState != nil {
					proc.Destroy()
				} else {
					proc.Release()
				}
			}
		}
	}()
	return &pool{pud}
}

func (*pool) private() {}

func (p *pool) Close() {
	p.puddle.Close()
}

func newWorker(ctx context.Context) (_ worker, err error) {
	tracer := trace.SpanFromContext(ctx).TracerProvider().Tracer("")
	ctx, span := tracer.Start(ctx, "jsonnetsecure.newWorker")
	defer otelx.End(span, &err)

	path, _ := ctx.Value(contextValuePath).(string)
	if path == "" {
		return worker{}, errors.New("newWorker: missing binary path in context")
	}
	args, _ := ctx.Value(contextValueArgs).([]string)
	cmd := exec.Command(path, append(args, "-0")...)
	cmd.Env = []string{"GOMAXPROCS=1"}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return worker{}, errors.Wrap(err, "newWorker: failed to create stdin pipe")
	}

	in := make(chan []byte)
	go func(c <-chan []byte) {
		for input := range c {
			if _, err := stdin.Write(append(input, 0)); err != nil {
				stdin.Close()
				return
			}
		}
	}(in)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return worker{}, errors.Wrap(err, "newWorker: failed to create stdout pipe")
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return worker{}, errors.Wrap(err, "newWorker: failed to create stderr pipe")
	}
	if err := cmd.Start(); err != nil {
		return worker{}, errors.Wrap(err, "newWorker: failed to start process")
	}

	scan := func(c chan<- string, r io.Reader) {
		defer close(c)
		scanner := bufio.NewScanner(r)
		scanner.Split(splitNull)
		for scanner.Scan() {
			c <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			c <- "ERROR: scan: " + err.Error()
		}
	}
	out := make(chan string)
	go scan(out, stdout)
	errs := make(chan string)
	go scan(errs, stderr)

	return worker{
		cmd:    cmd,
		stdin:  in,
		stdout: out,
		stderr: errs,
	}, nil
}

func (w worker) destroy() {
	close(w.stdin)
	w.cmd.Process.Kill()
}

func (w worker) eval(ctx context.Context, processParams []byte) (output string, err error) {
	tracer := trace.SpanFromContext(ctx).TracerProvider().Tracer("")
	ctx, span := tracer.Start(ctx, "jsonnetsecure.worker.eval",
		trace.WithAttributes(attribute.Int("cmd.Process.Pid", w.cmd.Process.Pid)))
	defer otelx.End(span, &err)

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case w.stdin <- processParams:
		break
	}

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case output := <-w.stdout:
		return output, nil
	case err := <-w.stderr:
		return "", errors.New(err)
	}
}

func (vm *processPoolVM) EvaluateAnonymousSnippet(filename string, snippet string) (_ string, err error) {
	tracer := trace.SpanFromContext(vm.ctx).TracerProvider().Tracer("")
	ctx, span := tracer.Start(vm.ctx, "jsonnetsecure.processPoolVM.EvaluateAnonymousSnippet", trace.WithAttributes(attribute.String("filename", filename)))
	defer otelx.End(span, &err)

	// TODO: maybe leave the timeout to the caller?
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	params := vm.params
	params.Filename = filename
	params.Snippet = snippet
	pp, err := json.Marshal(params)
	if err != nil {
		return "", errors.Wrap(err, "jsonnetsecure: marshal")
	}

	ctx = context.WithValue(ctx, contextValuePath, vm.path)
	ctx = context.WithValue(ctx, contextValueArgs, vm.args)
	worker, err := vm.pool.puddle.Acquire(ctx)
	if err != nil {
		return "", errors.Wrap(err, "jsonnetsecure: acquire")
	}

	result, err := worker.Value().eval(ctx, pp)
	if err != nil {
		worker.Destroy()
		return "", errors.Wrap(err, "jsonnetsecure: eval")
	} else {
		worker.Release()
	}

	if strings.HasPrefix(result, "ERROR: ") {
		return "", errors.New("jsonnetsecure: " + result)
	}

	return result, nil
}

func NewProcessPoolVM(opts *vmOptions) VM {
	ctx := opts.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return &processPoolVM{
		path: opts.jsonnetBinaryPath,
		args: opts.args,
		ctx:  ctx,
		pool: opts.pool,
	}
}

func (vm *processPoolVM) ExtCode(key string, val string) {
	vm.params.ExtCodes = append(vm.params.ExtCodes, kv{key, val})
}

func (vm *processPoolVM) ExtVar(key string, val string) {
	vm.params.ExtVars = append(vm.params.ExtVars, kv{key, val})
}

func (vm *processPoolVM) TLACode(key string, val string) {
	vm.params.TLACodes = append(vm.params.TLACodes, kv{key, val})
}

func (vm *processPoolVM) TLAVar(key string, val string) {
	vm.params.TLAVars = append(vm.params.TLAVars, kv{key, val})
}
