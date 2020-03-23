package viperx

import (
	"github.com/ory/viper"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"
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

const serviceName = "Test"

func TestWatchAndValidateViper(t *testing.T) {
	schema, err := ioutil.ReadFile("./stub/config.schema.json")
	require.NoError(t, err)

	t.Run("case=rejects not validating changes", func(t *testing.T) {
		config := `dsn: memory
foo: bar
`
		configFile, err := ioutil.TempFile("", "config-*.yaml")
		require.NoError(t, err)
		_, err = io.WriteString(configFile, config)
		require.NoError(t, err)
		require.NoError(t, configFile.Sync())
		t.Cleanup(func() {
			require.NoError(t, os.Remove(configFile.Name()))
		})
		l := logrus.New()
		l.ExitFunc = failOnExit(t)
		hook := test.NewLocal(l)
		viper.Reset()
		viper.SetConfigFile(configFile.Name())
		require.NoError(t, viper.ReadInConfig())

		WatchAndValidateViper(l, schema, serviceName, []string{})
		assert.Equal(t, []*logrus.Entry{}, hook.AllEntries())
		assert.Equal(t, "memory", viper.Get("dsn"))
		assert.Equal(t, "bar", viper.Get("foo"))

		_, err = configFile.Seek(0, 0)
		require.NoError(t, err)
		_, err = io.WriteString(configFile, strings.Replace(config, "bar", "not bar", 1))
		require.NoError(t, configFile.Sync())

		// viper needs some time to read the file
		entries := hook.AllEntries()
		for ; len(entries) < 2; entries = hook.AllEntries() {
		}
		require.Equal(t, 2, len(entries), "%+v", entries)
		assert.Equal(t, "The configuration has changed and was reloaded.", entries[0].Message)
		assert.Equal(t, "The changed configuration is invalid and could not be loaded. Rolling back to the last working configuration revision. Please address the validation errors before restarting Test.", entries[1].Message)
		assert.Equal(t, "memory", viper.Get("dsn"))
		assert.Equal(t, "bar", viper.Get("foo"))
	})

	t.Run("case=rejects to update immutable", func(t *testing.T) {
		config := `dsn: memory
foo: bar
`
		configFile, err := ioutil.TempFile("", "config-*.yaml")
		require.NoError(t, err)
		_, err = io.WriteString(configFile, config)
		require.NoError(t, err)
		require.NoError(t, configFile.Sync())
		t.Cleanup(func() {
			require.NoError(t, os.Remove(configFile.Name()))
		})
		l := logrus.New()
		l.ExitFunc = failOnExit(t)
		hook := test.NewLocal(l)
		viper.Reset()
		viper.SetConfigFile(configFile.Name())
		require.NoError(t, viper.ReadInConfig())

		WatchAndValidateViper(l, schema, serviceName, []string{"dsn"})
		assert.Equal(t, []*logrus.Entry{}, hook.AllEntries())
		val, err := viper.GetE("dsn")
		assert.NoError(t, err)
		assert.Equal(t, "memory", val)
		assert.Equal(t, "bar", viper.Get("foo"))

		_, err = configFile.Seek(0, 0)
		require.NoError(t, err)
		_, err = io.WriteString(configFile, strings.Replace(config, "memory", "some database", 1))
		require.NoError(t, configFile.Sync())

		time.Sleep(time.Second)
		// viper needs some time to read the file
		//entries := hook.AllEntries()
		//for ; len(entries) < 2; entries = hook.AllEntries() {}
		//require.Equal(t, 2, len(entries), "%+v", entries)
		//assert.Equal(t, "The configuration has changed and was reloaded.", entries[0].Message)
		//assert.Equal(t, "A configuration value marked as immutable has changed. Rolling back to the last working configuration revision. To reload the values please restart Test.", entries[1].Message)
		val, err = viper.GetE("dsn")
		assert.NoError(t, err)
		assert.Equal(t, "memory", val)
		assert.Equal(t, "bar", viper.Get("foo"))
	})

	t.Run("case=runs without validation errors", func(t *testing.T) {
		viper.Reset()
		l := logrus.New()
		l.ExitFunc = failOnExit(t)
		hook := test.NewLocal(l)

		viper.Set("dsn", "some string")
		viper.Set("foo", "bar")

		WatchAndValidateViper(l, schema, serviceName, []string{})

		assert.Equal(t, []*logrus.Entry{}, hook.AllEntries())
		assert.Equal(t, "some string", viper.Get("dsn"))
		assert.Equal(t, "bar", viper.Get("foo"))
	})

	t.Run("case=exits with validation errors", func(t *testing.T) {
		viper.Reset()
		l := logrus.New()
		l.ExitFunc = expectExit(t)
		hook := test.NewLocal(l)

		viper.Set("foo", "not bar")
		viper.Set("dsn", 0)

		WatchAndValidateViper(l, schema, serviceName, []string{})

		entries := hook.AllEntries()
		require.Equal(t, 1, len(entries))
		assert.Equal(t, "validation failed", entries[0].Data["[config_key=#]"])
		assert.Equal(t, "expected string, but got number", entries[0].Data["[config_key=dsn]"])
		assert.Equal(t, "value must be \"bar\"", entries[0].Data["[config_key=foo]"])
	})
}
