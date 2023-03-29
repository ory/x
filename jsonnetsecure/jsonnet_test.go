// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package jsonnetsecure

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/google/go-jsonnet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestSecureVM(t *testing.T) {
	for _, optCase := range []struct {
		name string
		opts []Option
	}{
		{"none", []Option{}},
		{"process vm", []Option{
			WithProcessIsolatedVM(context.Background()),
			WithJsonnetBinary(JsonnetTestBinary(t)),
		}},
	} {
		t.Run("options="+optCase.name, func(t *testing.T) {
			for i, contents := range []string{
				"local contents = importstr 'jsonnet.go'; { contents: contents }",
				"local contents = import 'stub/import.jsonnet'; { contents: contents }",
			} {
				t.Run(fmt.Sprintf("case=%d", i), func(t *testing.T) {
					vm := MakeSecureVM(optCase.opts...)
					result, err := vm.EvaluateAnonymousSnippet("test", contents)
					require.Error(t, err, "%s", result)
				})
			}
		})
	}

	// Test that all VM behave the same for sane input
	t.Run("suite=feature parity", func(t *testing.T) {
		t.Run("case=simple input", func(t *testing.T) {
			// from https://jsonnet.org/learning/tutorial.html
			snippet := `
/* A C-style comment. */
# A Python-style comment.
{
  cocktails: {
    // Ingredient quantities are in fl oz.
    'Tom Collins': {
      ingredients: [
        { kind: "Farmer's Gin", qty: 1.5 },
        { kind: 'Lemon', qty: 1 },
        { kind: 'Simple Syrup', qty: 0.5 },
        { kind: 'Soda', qty: 2 },
        { kind: 'Angostura', qty: 'dash' },
      ],
      garnish: 'Maraschino Cherry',
      served: 'Tall',
      description: |||
        The Tom Collins is essentially gin and
        lemonade.  The bitters add complexity.
      |||,
    },
    Manhattan: {
      ingredients: [
        { kind: 'Rye', qty: 2.5 },
        { kind: 'Sweet Red Vermouth', qty: 1 },
        { kind: 'Angostura', qty: 'dash' },
      ],
      garnish: 'Maraschino Cherry',
      served: 'Straight Up',
      description: @'A clear \ red drink.',
    },
  },
}`
			assertEqualVMOutput(t, func(factory func(t *testing.T) VM) string {
				vm := factory(t)
				out, err := vm.EvaluateAnonymousSnippet("test", snippet)
				assert.NoError(t, err)
				return out
			})
		})

		t.Run("case=ext variables", func(t *testing.T) {
			assertEqualVMOutput(t, func(factory func(t *testing.T) VM) string {
				vm := factory(t)
				vm.ExtVar("one", "1")
				vm.ExtVar("two", "2")
				vm.ExtCode("bool", "true")
				vm.TLAVar("oneArg", "1")
				vm.TLAVar("twoArg", "2")
				vm.TLACode("boolArg", "false")
				out, err := vm.EvaluateAnonymousSnippet(
					"test",
					`function (oneArg, twoArg, boolArg) {
						one: std.extVar("one"), two: std.extVar("two"), bool: std.extVar("bool"),
						oneTLA: oneArg, twoTLA: twoArg, boolTLA: boolArg,
					}`)
				assert.NoError(t, err)
				return out
			})
		})
	})

	t.Run("case=process isolation", func(t *testing.T) {
		snippet := "local f(x) = if x == 0 then [] else [f(x - 1), f(x - 1)]; f(100)"
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		vm := MakeSecureVM(
			WithProcessIsolatedVM(ctx),
			WithJsonnetBinary(JsonnetTestBinary(t)),
		)
		result, err := vm.EvaluateAnonymousSnippet("test", snippet)
		require.Error(t, err)

		// Error is either context.DeadlineExceeded or exec.ExitError, depending on whether
		// the process was already stared. We check for both to avoid flakes (both are fine).
		if errors.Is(err, context.DeadlineExceeded) {
			return
		}
		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr, result)
		assert.Equal(t, exitErr.ProcessState.ExitCode(), -1)
	})

	t.Run("case=importbin", func(t *testing.T) {
		// importbin does not exist in the current version, but is already merged on the main branch:
		// https://github.com/google/go-jsonnet/commit/856bd58872418eee1cede0badea5b7b462c429eb
		vm := MakeSecureVM()
		result, err := vm.EvaluateAnonymousSnippet(
			"test",
			"local contents = importbin 'stub/import.jsonnet'; { contents: contents }")
		require.Error(t, err, "%s", result)
	})
}

func standardVM(t *testing.T) VM { return jsonnet.MakeVM() }
func secureVM(t *testing.T) VM   { return MakeSecureVM() }
func processVM(t *testing.T) VM {
	return MakeSecureVM(
		WithProcessIsolatedVM(context.Background()),
		WithJsonnetBinary(JsonnetTestBinary(t)))
}
func assertEqualVMOutput(t *testing.T, run func(factory func(t *testing.T) VM) string) {
	t.Helper()

	expectedOut := run(standardVM)
	secureOut := run(secureVM)
	processOut := run(processVM)

	assert.Equal(t, expectedOut, secureOut, "secure output incorrect")
	assert.Equal(t, expectedOut, processOut, "process output incorrect")
}

func TestCreateMultipleProcessVMs(t *testing.T) {
	ctx := context.Background()
	wg := new(errgroup.Group)

	for i := 0; i < 100; i++ {
		wg.Go(func() error {
			vm := MakeSecureVM(
				WithProcessIsolatedVM(ctx),
				WithJsonnetBinary(JsonnetTestBinary(t)),
			)
			_, err := vm.EvaluateAnonymousSnippet("test", "{a:1}")

			return err
		})
	}

	require.NoError(t, wg.Wait())
}

func BenchmarkIsolatedVM(b *testing.B) {
	snippet := "{a:1}"
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	vm := MakeSecureVM(
		WithProcessIsolatedVM(ctx),
		WithJsonnetBinary(JsonnetTestBinary(b)),
	)

	for i := 0; i < b.N; i++ {
		_, err := vm.EvaluateAnonymousSnippet("test", snippet)
		if err != nil {
			require.NoError(b, err)
		}
	}
}

func BenchmarkRegularVM(b *testing.B) {
	snippet := "{a:1}"
	vm := MakeSecureVM()

	for i := 0; i < b.N; i++ {
		_, err := vm.EvaluateAnonymousSnippet("test", snippet)
		if err != nil {
			require.NoError(b, err)
		}
	}
}
