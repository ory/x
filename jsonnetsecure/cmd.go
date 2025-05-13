// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package jsonnetsecure

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"runtime"
	"syscall"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	MiB uint64 = 1024 * 1024
	// Generous limit including the peak memory allocated by the Go runtime, the Jsonnet VM,
	// and the Jsonnet script.
	// This number was acquired by running:
	// `echo -n '{"Snippet":"std.repeat(\"a\", 1000)"}' | rusage ./kratos jsonnet > /dev/null
	// which outputs among other things: `ballooned to 45,088kb in size` (i.e. ~45 MiB).
	// Thus we raise this number a bit for safety and call it a day.
	memoryLimit = 64 * MiB
)

func NewJsonnetCmd() *cobra.Command {
	var null bool
	cmd := &cobra.Command{
		Use:    "jsonnet",
		Short:  "Run Jsonnet as a CLI command",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// - macos: `setrlimit(2)` with `RLIMIT_AS` seems broken on macOS and its behavior
			// varies between major versions. Also there is not really a use case for macOS server-side, so we do not
			// bother.
			// - windows: This syscall is Unix specific so not available.
			if !(runtime.GOOS == "windows" || runtime.GOOS == "darwin") {
				limit := syscall.Rlimit{
					Cur: memoryLimit,
					Max: memoryLimit,
				}
				err := syscall.Setrlimit(syscall.RLIMIT_AS, &limit)
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to set memory limit %d: %+v\n", memoryLimit, err)
					// It could fail because current limits are lower than what we tried to set,
					// so we still continue in this case.
				}
			}

			if null {
				return scan(cmd.OutOrStdout(), cmd.InOrStdin())
			}

			input, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return errors.Wrap(err, "failed to read from stdin")
			}

			json, err := eval(input)
			if err != nil {
				return errors.Wrap(err, "failed to evaluate jsonnet")
			}

			if _, err := io.WriteString(cmd.OutOrStdout(), json); err != nil {
				return errors.Wrap(err, "failed to write json output")
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&null, "null", "0", false,
		`Read multiple snippets and parameters from stdin separated by null bytes.
Output will be in the same order as inputs, separated by null bytes.
Evaluation errors will also be reported to stdout, separated by null bytes.
Non-recoverable errors are written to stderr and the program will terminate with a non-zero exit code.`)

	return cmd
}

func scan(w io.Writer, r io.Reader) error {
	scanner := bufio.NewScanner(r)
	scanner.Split(splitNull)
	for scanner.Scan() {
		json, err := eval(scanner.Bytes())
		if err != nil {
			json = fmt.Sprintf("ERROR: %s", err)
		}
		if _, err := fmt.Fprintf(w, "%s%c", json, 0); err != nil {
			return errors.Wrap(err, "failed to write json output")
		}
	}
	return errors.Wrap(scanner.Err(), "failed to read from stdin")
}

func eval(input []byte) (json string, err error) {
	var params processParameters
	if err := params.Decode(input); err != nil {
		return "", err
	}

	vm := MakeSecureVM()

	for _, it := range params.ExtCodes {
		vm.ExtCode(it.Key, it.Value)
	}
	for _, it := range params.ExtVars {
		vm.ExtVar(it.Key, it.Value)
	}
	for _, it := range params.TLACodes {
		vm.TLACode(it.Key, it.Value)
	}
	for _, it := range params.TLAVars {
		vm.TLAVar(it.Key, it.Value)
	}

	return vm.EvaluateAnonymousSnippet(params.Filename, params.Snippet)
}
