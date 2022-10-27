package configx

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/x/logrusx"
	"github.com/ory/x/watcherx"
)

var ctx = context.Background()

func tmpConfigFile(t *testing.T, dsn, foo string) *os.File {
	config := fmt.Sprintf("dsn: %s\nfoo: %s\n", dsn, foo)

	tdir := filepath.Join(os.TempDir(), strconv.FormatInt(time.Now().UnixNano(), 10))
	require.NoError(t,
		os.MkdirAll(tdir, // DO NOT CHANGE THIS: https://github.com/fsnotify/fsnotify/issues/340
			os.ModePerm))
	configFile, err := os.CreateTemp(tdir, "config-*.yml")
	require.NoError(t, err)
	_, err = io.WriteString(configFile, config)
	require.NoError(t, err)
	require.NoError(t, configFile.Sync())
	t.Cleanup(func() {
		fmt.Printf("removing %s\n", configFile.Name())
		_ = os.Remove(configFile.Name())
	})

	return configFile
}

func updateConfigFile(t *testing.T, c <-chan struct{}, configFile *os.File, dsn, foo, bar string) {
	config := fmt.Sprintf(`dsn: %s
foo: %s
bar: %s`, dsn, foo, bar)

	_, err := configFile.Seek(0, 0)
	require.NoError(t, err)
	require.NoError(t, configFile.Truncate(0))
	_, err = io.WriteString(configFile, config)
	require.NoError(t, err)
	require.NoError(t, configFile.Sync())
	<-c // Wait for changes to propagate
	time.Sleep(time.Millisecond)
}

func lsof(t *testing.T, file string) string {
	if runtime.GOOS == "windows" {
		return ""
	}
	var b, be bytes.Buffer
	c := exec.Command("lsof", "-n")
	c.Stdout = &b
	c.Stderr = &be
	require.NoError(t, c.Run(), "stderr says: %s %s", be.String(), b.String())
	var out string
	scanner := bufio.NewScanner(&b)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.Contains(text, file) {
			out += text + "\n"
			break
		}
	}
	require.NoError(t, scanner.Err())
	return out
}

func checkLsof(t *testing.T, file string) string {
	if runtime.GOOS == "windows" {
		return ""
	}

	var b bytes.Buffer
	c := exec.Command("bash", "-c", "lsof -n | grep '"+file+"' | wc -l")
	c.Stdout = &b
	require.NoError(t, c.Run(), c.String())
	return b.String()
}

func compareLsof(t *testing.T, file, atStart, expected string) {
	var actual string
	for i := 0; i < 5; i++ {
		actual = checkLsof(t, file)
		if expected == actual {
			break
		}
	}

	e, err := strconv.ParseInt(strings.TrimSpace(expected), 10, 64)
	require.NoError(t, err)
	a, err := strconv.ParseInt(strings.TrimSpace(actual), 10, 64)
	require.NoError(t, err)

	const deviation = 6
	assert.True(t, e < a+deviation && e > a-deviation, "\n\t%s\n\t%s", atStart, lsof(t, file))
}

func TestReload(t *testing.T) {
	setup := func(t *testing.T, cf *os.File, c chan<- struct{}, modifiers ...OptionModifier) (*Provider, *logrusx.Logger) {
		l := logrusx.New("configx", "test")
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		modifiers = append(modifiers,
			WithLogrusWatcher(l),
			WithLogger(l),
			AttachWatcher(func(event watcherx.Event, err error) {
				t.Logf("Received event: %+v error: %+v", event, err)
				c <- struct{}{}
			}),
			WithContext(ctx),
		)
		p, err := newKoanf(ctx, "./stub/watch/config.schema.json", []string{cf.Name()}, modifiers...)
		require.NoError(t, err)
		return p, l
	}

	t.Run("case=rejects not validating changes", func(t *testing.T) {
		configFile := tmpConfigFile(t, "memory", "bar")
		defer configFile.Close()
		c := make(chan struct{})
		p, l := setup(t, configFile, c)
		hook := test.NewLocal(l.Entry.Logger)

		atStart := checkLsof(t, configFile.Name())
		lsofAtStart := lsof(t, configFile.Name())

		assert.Equal(t, []*logrus.Entry{}, hook.AllEntries())
		assert.Equal(t, "memory", p.String("dsn"))
		assert.Equal(t, "bar", p.String("foo"))

		updateConfigFile(t, c, configFile, "memory", "not bar", "bar")

		entries := hook.AllEntries()
		require.False(t, len(entries) > 4, "%+v", entries) // should be 2 but addresses flake https://github.com/ory/x/runs/2332130952

		assert.Equal(t, "A change to a configuration file was detected.", entries[0].Message)
		assert.Equal(t, "The changed configuration is invalid and could not be loaded. Rolling back to the last working configuration revision. Please address the validation errors before restarting the process.", entries[1].Message)

		assert.Equal(t, "memory", p.String("dsn"))
		assert.Equal(t, "bar", p.String("foo"))

		// but it is still watching the files
		updateConfigFile(t, c, configFile, "memory", "bar", "baz")
		assert.Equal(t, "baz", p.String("bar"))

		time.Sleep(time.Millisecond * 250)

		compareLsof(t, configFile.Name(), lsofAtStart, atStart)
	})

	t.Run("case=rejects to update immutable", func(t *testing.T) {
		configFile := tmpConfigFile(t, "memory", "bar")
		defer configFile.Close()
		c := make(chan struct{})
		p, l := setup(t, configFile, c,
			WithImmutables("dsn"))
		hook := test.NewLocal(l.Entry.Logger)

		atStart := checkLsof(t, configFile.Name())
		lsofAtStart := lsof(t, configFile.Name())

		assert.Equal(t, []*logrus.Entry{}, hook.AllEntries())
		assert.Equal(t, "memory", p.String("dsn"))
		assert.Equal(t, "bar", p.String("foo"))

		updateConfigFile(t, c, configFile, "some db", "bar", "baz")

		entries := hook.AllEntries()
		require.False(t, len(entries) > 4, "%+v", entries) // should be 2 but addresses flake https://github.com/ory/x/runs/2332130952
		assert.Equal(t, "A change to a configuration file was detected.", entries[0].Message)
		assert.Equal(t, "A configuration value marked as immutable has changed. Rolling back to the last working configuration revision. To reload the values please restart the process.", entries[1].Message)
		assert.Equal(t, "memory", p.String("dsn"))
		assert.Equal(t, "bar", p.String("foo"))

		// but it is still watching the files
		updateConfigFile(t, c, configFile, "memory", "bar", "baz")
		assert.Equal(t, "baz", p.String("bar"))

		compareLsof(t, configFile.Name(), lsofAtStart, atStart)
	})

	t.Run("case=runs without validation errors", func(t *testing.T) {
		configFile := tmpConfigFile(t, "some string", "bar")
		defer configFile.Close()
		c := make(chan struct{})
		p, l := setup(t, configFile, c)
		hook := test.NewLocal(l.Entry.Logger)

		assert.Equal(t, []*logrus.Entry{}, hook.AllEntries())
		assert.Equal(t, "some string", p.String("dsn"))
		assert.Equal(t, "bar", p.String("foo"))
	})

	t.Run("case=runs and reloads", func(t *testing.T) {
		configFile := tmpConfigFile(t, "some string", "bar")
		defer configFile.Close()
		c := make(chan struct{})
		p, l := setup(t, configFile, c)
		hook := test.NewLocal(l.Entry.Logger)

		assert.Equal(t, []*logrus.Entry{}, hook.AllEntries())
		assert.Equal(t, "some string", p.String("dsn"))
		assert.Equal(t, "bar", p.String("foo"))

		updateConfigFile(t, c, configFile, "memory", "bar", "baz")
		assert.Equal(t, "baz", p.String("bar"))
	})

	t.Run("case=has with validation errors", func(t *testing.T) {
		configFile := tmpConfigFile(t, "some string", "not bar")
		defer configFile.Close()
		l := logrusx.New("", "")
		hook := test.NewLocal(l.Entry.Logger)

		var b bytes.Buffer
		_, err := newKoanf(ctx, "./stub/watch/config.schema.json", []string{configFile.Name()},
			WithStandardValidationReporter(&b),
			WithLogrusWatcher(l),
		)
		require.Error(t, err)

		entries := hook.AllEntries()
		require.Equal(t, 0, len(entries))
		assert.Equal(t, "The configuration contains values or keys which are invalid:\nfoo: not bar\n     ^-- value must be \"bar\"\n\n", b.String())
	})

	t.Run("case=is not leaking open files", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip()
		}

		configFile := tmpConfigFile(t, "some string", "bar")
		defer configFile.Close()
		c := make(chan struct{})
		p, _ := setup(t, configFile, c)

		atStart := checkLsof(t, configFile.Name())
		lsofAtStart := lsof(t, configFile.Name())
		for i := 0; i < 30; i++ {
			t.Run(fmt.Sprintf("iteration=%d", i), func(t *testing.T) {
				expected := []string{"foo", "bar", "baz"}[i%3]
				updateConfigFile(t, c, configFile, "memory", "bar", expected)
				require.EqualValues(t, atStart, checkLsof(t, configFile.Name()))
				require.EqualValues(t, expected, p.String("bar"))
			})
		}

		compareLsof(t, configFile.Name(), lsofAtStart, atStart)

		atStartNum, err := strconv.ParseInt(strings.TrimSpace(atStart), 10, 32)
		require.NoError(t, err)
		require.True(t, atStartNum < 20, "should not be unreasonably high: %s\n\t%s", atStartNum, lsofAtStart)
	})

	t.Run("case=callback can use the provider to get the new value", func(t *testing.T) {
		dsn := "old"

		f := tmpConfigFile(t, dsn, "bar")
		c := make(chan struct{})

		var p *Provider
		p, _ = setup(t, f, c, AttachWatcher(func(watcherx.Event, error) {
			dsn = p.String("dsn")
		}))

		// change dsn
		updateConfigFile(t, c, f, "new", "bar", "bar")

		assert.Equal(t, "new", dsn)
	})
}
