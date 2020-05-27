package viperx

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/viper"

	"github.com/ory/x/logrusx"
)

func failOnExit(t *testing.T) func(i int) {
	return func(i int) {
		t.Fatalf("unexpectedly exited with code %d", i)
	}
}

func expectExit(t *testing.T) func(int) {
	return func(i int) {
		assert.NotEqual(t, 0, i)
	}
}

const productName = "Test"

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
		require.NoError(t, os.Remove(configFile.Name()))
	})

	return configFile
}

func updateConfigFile(t *testing.T, configFile *os.File, dsn, foo string) {
	config := fmt.Sprintf("dsn: %s\nfoo: %s\n", dsn, foo)

	_, err := configFile.Seek(0, 0)
	require.NoError(t, err)
	_, err = io.WriteString(configFile, config)
	require.NoError(t, configFile.Sync())
}

func setup(t *testing.T, exitFunc func(int), configFile *os.File) (*logrusx.Logger, *test.Hook) {
	l := logrusx.New("","")
	l.Entry.Logger.ExitFunc =exitFunc
	viper.Reset()

	if configFile != nil {
		viper.SetConfigFile(configFile.Name())
		require.NoError(t, viper.ReadInConfig())
	} else {
		t.Logf("Config file is nil")
	}

	return l, test.NewLocal(l.Entry.Logger)
}

func TestWatchAndValidateViper(t *testing.T) {
	schema, err := ioutil.ReadFile("./stub/config.schema.json")
	require.NoError(t, err)

	t.Run("case=rejects not validating changes", func(t *testing.T) {
		configFile := tmpConfigFile(t, "memory", "bar")

		l, hook := setup(t, failOnExit(t), configFile)

		WatchAndValidateViper(l, schema, productName, []string{})
		assert.Equal(t, []*logrus.Entry{}, hook.AllEntries())
		assert.Equal(t, "memory", viper.Get("dsn"))
		assert.Equal(t, "bar", viper.Get("foo"))

		updateConfigFile(t, configFile, "memory", "not bar")

		// viper needs some time to read the file
		entries := hook.AllEntries()
		for ; len(entries) < 2; entries = hook.AllEntries() {
			time.Sleep(time.Millisecond)
		}
		require.Equal(t, 2, len(entries))

		assert.Equal(t, "The configuration has changed and was reloaded.", entries[0].Message)
		assert.Equal(t, "The changed configuration is invalid and could not be loaded. Rolling back to the last working configuration revision. Please address the validation errors before restarting Test.", entries[1].Message)

		assert.Equal(t, "memory", viper.Get("dsn"))
		assert.Equal(t, "bar", viper.Get("foo"))
	})

	t.Run("case=rejects to update immutable", func(t *testing.T) {
		configFile := tmpConfigFile(t, "memory", "bar")

		l, hook := setup(t, failOnExit(t), configFile)

		WatchAndValidateViper(l, schema, productName, []string{"dsn"})
		assert.Equal(t, []*logrus.Entry{}, hook.AllEntries())
		assert.Equal(t, "memory", viper.Get("dsn"))
		assert.Equal(t, "bar", viper.Get("foo"))

		updateConfigFile(t, configFile, "some db", "bar")

		// viper needs some time to read the file
		entries := hook.AllEntries()
		for ; len(entries) < 2; entries = hook.AllEntries() {
		}
		require.Equal(t, 2, len(entries))
		assert.Equal(t, "The configuration has changed and was reloaded.", entries[0].Message)
		assert.Equal(t, "A configuration value marked as immutable has changed. Rolling back to the last working configuration revision. To reload the values please restart Test.", entries[1].Message)
		assert.Equal(t, "memory", viper.Get("dsn"))
		assert.Equal(t, "bar", viper.Get("foo"))
	})

	t.Run("case=runs without validation errors", func(t *testing.T) {
		l, hook := setup(t, failOnExit(t), nil)

		viper.Set("dsn", "some string")
		viper.Set("foo", "bar")

		WatchAndValidateViper(l, schema, productName, []string{})

		assert.Equal(t, []*logrus.Entry{}, hook.AllEntries())
		assert.Equal(t, "some string", viper.Get("dsn"))
		assert.Equal(t, "bar", viper.Get("foo"))
	})

	t.Run("case=exits with validation errors", func(t *testing.T) {
		l, hook := setup(t, expectExit(t), nil)

		viper.Set("foo", "not bar")
		viper.Set("dsn", 0)

		WatchAndValidateViper(l, schema, productName, []string{})

		entries := hook.AllEntries()
		require.Equal(t, 2, len(entries))
		assert.Equal(t, "The provided configuration is invalid and could not be loaded. Check the output below to understand why.", entries[0].Message)
		assert.Equal(t, "The services failed to start because the configuration is invalid. Check the output above for more details.", entries[1].Message)
	})
}
