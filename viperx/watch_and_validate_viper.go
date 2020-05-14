package viperx

import (
	"fmt"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"

	"github.com/ory/viper"
)

func WatchAndValidateViper(l logrus.FieldLogger, schema []byte, productName string, immutables []string) {
	if err := Validate("config.schema.json", schema); err != nil {
		l.WithField("config_file", viper.ConfigFileUsed()).Error("The provided configuration is invalid and could not be loaded. Check the output below to understand why.")
		_, _ = fmt.Fprintln(os.Stderr, "")
		PrintHumanReadableValidationErrors(os.Stderr, err)
		os.Exit(1)
	}

	AddWatcher(func(event fsnotify.Event) error {
		if err := Validate("config.schema.json", schema); err != nil {
			PrintHumanReadableValidationErrors(os.Stderr, err)
			l.Errorf("The changed configuration is invalid and could not be loaded. Rolling back to the last working configuration revision. Please address the validation errors before restarting %s.", productName)
			return ErrRollbackConfigurationChanges
		}
		return nil
	})

	WatchConfig(l, &WatchOptions{
		Immutables: immutables,
		OnImmutableChange: func(key string) {
			l.
				WithField("key", key).
				WithField("reset_to", fmt.Sprintf("%v", viper.Get(key))).
				Errorf("A configuration value marked as immutable has changed. Rolling back to the last working configuration revision. To reload the values please restart %s.", productName)
		},
	})
}
