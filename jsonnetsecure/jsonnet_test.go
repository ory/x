// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package jsonnetsecure

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-jsonnet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func ensureChildProcessStoppedEarly(t testing.TB, err error) {
	t.Helper()

	require.Error(t, err)
	// The actual string is OS-specific and our tests run on all major ones.
	// Additionally the child process may have stopped/been stopped for a variety of reasons,
	// depending on which limit was hit first.
	errStr := err.Error()
	require.True(t,
		// Killed by the parent or the OS (due to hitting the memory limit).
		strings.Contains(errStr, "reached limits") ||
			strings.Contains(errStr, "killed") ||

			// The Go runtime hit the memory limit and quit.
			strings.Contains(errStr, "cannot allocate memory") ||

			// Invalid input.
			strings.Contains(errStr, "encountered an error") ||
			// Timeout.
			strings.Contains(errStr, "deadline exceeded") ||
			// Too much output (this error comes from `bufio.Scanner` which has its own internal limit).
			strings.Contains(errStr, "token too long"),
		errStr,
	)

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		assert.NotEqual(t, exitErr.ProcessState.ExitCode(), 0)
	}
}

func TestSecureVM(t *testing.T) {
	testBinary := JsonnetTestBinary(t)

	for _, optCase := range []struct {
		name string
		opts []Option
	}{
		{"none", []Option{}},
		{"process vm", []Option{
			WithProcessIsolatedVM(context.Background()),
			WithJsonnetBinary(testBinary),
		}},
		{"process pool vm", []Option{
			WithProcessIsolatedVM(context.Background()),
			WithProcessPool(procPool),
			WithJsonnetBinary(testBinary),
		}},
	} {
		t.Run("options="+optCase.name, func(t *testing.T) {
			for i, contents := range []string{
				"local contents = importstr 'jsonnet.go'; { contents: contents }",
				"local contents = import 'stub/import.jsonnet'; { contents: contents }",
				`{user_id: ` + strings.Repeat("a", jsonnetErrLimit*5),
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

	t.Run("case=stack overflow", func(t *testing.T) {
		snippet := "local f(x) = if x == 0 then [] else [f(x - 1), f(x - 1)]; f(100)"
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		t.Cleanup(cancel)
		vm := MakeSecureVM(
			WithProcessIsolatedVM(ctx),
			WithJsonnetBinary(testBinary),
		)
		_, err := vm.EvaluateAnonymousSnippet("test", snippet)
		ensureChildProcessStoppedEarly(t, err)
	})

	t.Run("case=stack overflow pool", func(t *testing.T) {
		snippet := "local f(x) = if x == 0 then [] else [f(x - 1), f(x - 1)]; f(100)"
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		t.Cleanup(cancel)
		vm := MakeSecureVM(
			WithProcessIsolatedVM(ctx),
			WithJsonnetBinary(testBinary),
			WithProcessPool(procPool),
		)
		result, err := vm.EvaluateAnonymousSnippet("test", snippet)
		ensureChildProcessStoppedEarly(t, err)
		assert.Empty(t, result)
	})

	t.Run("case=stdout too lengthy", func(t *testing.T) {
		// This script outputs more than the limit.
		snippet := `{user_id: std.repeat("a", ` + strconv.FormatUint(jsonnetOutputLimit, 10) + `)}`
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		t.Cleanup(cancel)
		vm := MakeSecureVM(
			WithProcessIsolatedVM(ctx),
			WithJsonnetBinary(testBinary),
		)
		_, err := vm.EvaluateAnonymousSnippet("test", snippet)
		ensureChildProcessStoppedEarly(t, err)
	})

	t.Run("case=stdout too lengthy pool", func(t *testing.T) {
		// This script outputs more than the limit.
		snippet := `{user_id: std.repeat("a", ` + strconv.FormatUint(jsonnetOutputLimit, 10) + `)}`
		_, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		t.Cleanup(cancel)
		vm := MakeSecureVM(
			WithProcessPool(procPool),
			WithJsonnetBinary(testBinary),
		)
		_, err := vm.EvaluateAnonymousSnippet("test", snippet)
		ensureChildProcessStoppedEarly(t, err)
	})

	t.Run("case=stderr truncated", func(t *testing.T) {
		// Intentionally incorrect jsonnet syntax to trigger an error
		snippet := `{user_id: ` + strings.Repeat("a", jsonnetErrLimit*5)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		t.Cleanup(cancel)
		vm := MakeSecureVM(
			WithProcessIsolatedVM(ctx),
			WithJsonnetBinary(testBinary),
		)
		_, err := vm.EvaluateAnonymousSnippet("test", snippet)
		// Check that the stderr is truncated.
		// The jsonnet vm will print some stuff along the error so we need to acccount for that in the size.
		require.Error(t, err)
		require.Less(t, len(err.Error()), jsonnetErrLimit*2)
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

func standardVM(t *testing.T) VM {
	t.Helper()
	return jsonnet.MakeVM()
}

func secureVM(t *testing.T) VM {
	t.Helper()
	return MakeSecureVM()
}

func processVM(t *testing.T) VM {
	t.Helper()
	return MakeSecureVM(
		WithProcessIsolatedVM(context.Background()),
		WithJsonnetBinary(JsonnetTestBinary(t)))
}

func poolVM(t *testing.T) VM {
	t.Helper()
	pool := NewProcessPool(10)
	t.Cleanup(pool.Close)
	return MakeSecureVM(
		WithProcessIsolatedVM(context.Background()),
		WithProcessPool(pool),
		WithJsonnetBinary(JsonnetTestBinary(t)))
}

func assertEqualVMOutput(t *testing.T, run func(factory func(t *testing.T) VM) string) {
	t.Helper()

	expectedOut := run(standardVM)
	secureOut := run(secureVM)
	processOut := run(processVM)
	poolOut := run(poolVM)

	assert.Equal(t, expectedOut, secureOut, "secure output incorrect")
	assert.Equal(t, expectedOut, processOut, "process output incorrect")
	assert.Equal(t, expectedOut, poolOut, "pool output incorrect")
}

func TestStressTest(t *testing.T) {
	ctx := context.Background()
	wg := errgroup.Group{}
	// It's easy to overwhelm certain OSes with too many spawned processes at once.
	wg.SetLimit(8)
	testBinary := JsonnetTestBinary(t)

	count := 200
	type Case struct {
		snippet     string
		errExpected bool
	}

	cases := []Case{
		{snippet: `{a:1}`, errExpected: false},                       // Correct.
		{snippet: `{a: std.repeat("a",1000000)}`, errExpected: true}, // Correct but output is too lengthy.
		{snippet: `{a:`, errExpected: true},                          // Incorrect syntax (will print on stderr).
	}
	for i := range count {
		wg.Go(func() error {
			vm := MakeSecureVM(
				WithProcessIsolatedVM(ctx),
				WithJsonnetBinary(testBinary),
			)
			c := cases[i%len(cases)]
			_, err := vm.EvaluateAnonymousSnippet("test", c.snippet)

			// An error happened and was expected: ok.
			if c.errExpected {
				require.Error(t, err)
				return nil
			}

			// An error happened but none was expected.
			// Special case for macOS where data-races can happen with `kill(2)` where we
			// `kill(2)` the wrong process e.g. the one for a valid script.
			// We cannot avoid this issue so we simply swallow the error.
			// Other OSes have saner, race-free APIs e.g. https://www.man7.org/linux/man-pages/man2/pidfd_send_signal.2.html.
			if err != nil && runtime.GOOS == "darwin" && strings.Contains(err.Error(), "signal: killed") {
				return nil
			}

			if err != nil {
				t.Logf("err: i=%d case=%+v err=%+v", i, c, err)
			}
			return err
		})
	}

	require.NoError(t, wg.Wait())
}

func TestMain(m *testing.M) {
	procPool = NewProcessPool(runtime.GOMAXPROCS(0))
	defer procPool.Close()
	m.Run()
}

var (
	procPool Pool
	snippet  = "{a:std.extVar('a')}"
)

func BenchmarkIsolatedVM(b *testing.B) {
	binary := JsonnetTestBinary(b)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			vm := MakeSecureVM(
				WithProcessIsolatedVM(context.Background()),
				WithJsonnetBinary(binary),
			)
			i := rand.Int()
			vm.ExtCode("a", strconv.Itoa(i))
			res, err := vm.EvaluateAnonymousSnippet("test", snippet)
			require.NoError(b, err)
			require.JSONEq(b, fmt.Sprintf(`{"a": %d}`, i), res)
		}
	})
}

func BenchmarkProcessPoolVM(b *testing.B) {
	binary := JsonnetTestBinary(b)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			vm := MakeSecureVM(
				WithJsonnetBinary(binary),
				WithProcessPool(procPool),
			)
			i := rand.Int()
			vm.ExtCode("a", strconv.Itoa(i))
			res, err := vm.EvaluateAnonymousSnippet("test", snippet)
			require.NoError(b, err)
			require.JSONEq(b, fmt.Sprintf(`{"a": %d}`, i), res)
		}
	})
}

func BenchmarkRegularVM(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			vm := MakeSecureVM()
			i := rand.Int()
			vm.ExtCode("a", strconv.Itoa(i))
			res, err := vm.EvaluateAnonymousSnippet("test", snippet)
			require.NoError(b, err)
			require.JSONEq(b, fmt.Sprintf(`{"a": %d}`, i), res)
		}
	})
}

func BenchmarkReusableProcessVM(b *testing.B) {
	var (
		binary = JsonnetTestBinary(b)
		cmd    = exec.Command(binary, "-0")
		inputs = make(chan struct{})
		stderr strings.Builder
		eg     errgroup.Group
		count  int32 = 0
	)
	stdin, err := cmd.StdinPipe()
	require.NoError(b, err)
	stdout, err := cmd.StdoutPipe()
	require.NoError(b, err)
	cmd.Stderr = &stderr
	require.NoError(b, cmd.Start())

	b.Cleanup(func() {
		close(inputs)
		assert.NoError(b, stdin.Close())
		assert.NoError(b, eg.Wait())
		assert.NoError(b, cmd.Wait())
		assert.Empty(b, stderr.String())
	})

	eg.Go(func() error {
		scanner := bufio.NewScanner(stdout)
		scanner.Split(splitNull)
		for scanner.Scan() {
			c := atomic.AddInt32(&count, 1)
			require.JSONEq(b, fmt.Sprintf(`{"a": %d}`, c), scanner.Text())
		}
		return scanner.Err()
	})

	eg.Go(func() error {
		a := 1
		for range inputs {
			pp := processParameters{Snippet: snippet, ExtCodes: []kv{{"a", strconv.Itoa(a)}}}
			a++
			require.NoError(b, pp.EncodeTo(stdin))
			_, err := stdin.Write([]byte{0})
			require.NoError(b, err)
		}
		return nil
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			inputs <- struct{}{}
		}
	})
	for atomic.LoadInt32(&count) != int32(b.N) {
		time.Sleep(1 * time.Millisecond)
	}
}
