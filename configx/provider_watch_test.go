package configx

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
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

func tmpConfigFile(t *testing.T, dsn, foo string) *os.File {
	config := fmt.Sprintf("dsn: %s\nfoo: %s\n", dsn, foo)

	tdir := os.TempDir() + "/" + strconv.Itoa(time.Now().Nanosecond())
	require.NoError(t,
		os.MkdirAll(tdir, // DO NOT CHANGE THIS: https://github.com/fsnotify/fsnotify/issues/340
			os.ModePerm))
	configFile, err := ioutil.TempFile(tdir, "config-*.yml")
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
	require.NoError(t, configFile.Sync())
	<-c // Wait for changes to propagate
}

func checkLsof(t *testing.T, file string) string {
	if runtime.GOOS == "windows" {
		return ""
	}
	var b bytes.Buffer
	c := exec.Command("bash", "-c", "lsof -n | grep "+file+" | wc -l")
	c.Stdout = &b
	require.NoError(t, c.Run())
	return b.String()
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
		p, err := newKoanf("./stub/watch/config.schema.json", []string{cf.Name()}, modifiers...)
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

		assert.Equal(t, []*logrus.Entry{}, hook.AllEntries())
		assert.Equal(t, "memory", p.String("dsn"))
		assert.Equal(t, "bar", p.String("foo"))

		updateConfigFile(t, c, configFile, "memory", "not bar", "bar")

		entries := hook.AllEntries()
		require.Equal(t, 2, len(entries))

		assert.Equal(t, "A change to a configuration file was detected.", entries[0].Message)
		assert.Equal(t, "The changed configuration is invalid and could not be loaded. Rolling back to the last working configuration revision. Please address the validation errors before restarting the process.", entries[1].Message)

		assert.Equal(t, "memory", p.String("dsn"))
		assert.Equal(t, "bar", p.String("foo"))

		// but it is still watching the files
		updateConfigFile(t, c, configFile, "memory", "bar", "baz")
		assert.Equal(t, "baz", p.String("bar"))

		atEnd := checkLsof(t, configFile.Name())
		require.EqualValues(t, atStart, atEnd)
	})

	t.Run("case=rejects to update immutable", func(t *testing.T) {
		configFile := tmpConfigFile(t, "memory", "bar")
		defer configFile.Close()
		c := make(chan struct{})
		p, l := setup(t, configFile, c,
			WithImmutables("dsn"))
		hook := test.NewLocal(l.Entry.Logger)

		atStart := checkLsof(t, configFile.Name())

		assert.Equal(t, []*logrus.Entry{}, hook.AllEntries())
		assert.Equal(t, "memory", p.String("dsn"))
		assert.Equal(t, "bar", p.String("foo"))

		updateConfigFile(t, c, configFile, "some db", "bar", "baz")

		entries := hook.AllEntries()
		require.Equal(t, 2, len(entries))
		assert.Equal(t, "A change to a configuration file was detected.", entries[0].Message)
		assert.Equal(t, "A configuration value marked as immutable has changed. Rolling back to the last working configuration revision. To reload the values please restart the process.", entries[1].Message)
		assert.Equal(t, "memory", p.String("dsn"))
		assert.Equal(t, "bar", p.String("foo"))

		// but it is still watching the files
		updateConfigFile(t, c, configFile, "memory", "bar", "baz")
		assert.Equal(t, "baz", p.String("bar"))

		atEnd := checkLsof(t, configFile.Name())
		require.EqualValues(t, atStart, atEnd)
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
		_, err := newKoanf("./stub/watch/config.schema.json", []string{configFile.Name()},
			WithStandardValidationReporter(&b),
			WithLogrusWatcher(l),
		)
		require.Error(t, err)

		entries := hook.AllEntries()
		require.Equal(t, 0, len(entries))
		assert.Equal(t, "The configuration contains values or keys which are invalid:\nfoo: not bar\n     ^-- value must be \"bar\"\n\n", b.String())
	})

	t.Run("case=is not leaking open files", func(t *testing.T) {
		configFile := tmpConfigFile(t, "some string", "bar")
		defer configFile.Close()
		c := make(chan struct{})
		p, _ := setup(t, configFile, c)

		atStart := checkLsof(t, configFile.Name())
		for i := 0; i < 30; i++ {
			t.Run(fmt.Sprintf("iteration=%d", i), func(t *testing.T) {
				expected := []string{"foo", "bar", "baz"}[i%3]
				updateConfigFile(t, c, configFile, "memory", "bar", expected)
				require.EqualValues(t, atStart, checkLsof(t, configFile.Name()))
				require.EqualValues(t, expected, p.String("bar"))
			})
		}

		atEnd := checkLsof(t, configFile.Name())
		require.EqualValues(t, atStart, atEnd)

		atStartNum, err := strconv.ParseInt(strings.TrimSpace(atStart), 10, 32)
		require.NoError(t, err)
		require.True(t, atStartNum < 20, "should not be unreasonably high: %s", atStartNum)
	})
}
