package viperx

import (
	"fmt"
	"os"

	"github.com/fsnotify/fsnotify"

	"github.com/ory/viper"

	"github.com/ory/x/logrusx"
)

func WatchAndValidateViper(l *logrusx.Logger, schema []byte, productName string, immutables []string, sensitiveDumpConfigDir string) {
	if !l.LeakSensitiveData() && sensitiveDumpConfigDir != "" {
		l.Warn("The configuration is not going to be dumped as it contains sensitive information. To enable config dumping you have to enable sensitive logging.")
	}
	if err := Validate("config.schema.json", schema); err != nil {
		l.WithField("config_file", viper.ConfigFileUsed()).Error("The provided configuration is invalid and could not be loaded. Check the output below to understand why.")
		_, _ = fmt.Fprintln(os.Stderr, "")
		PrintHumanReadableValidationErrors(os.Stderr, err)
		l.Fatalf("The services failed to start because the configuration is invalid. Check the output above for more details.")
	}
	if l.LeakSensitiveData() && sensitiveDumpConfigDir != "" {
		if err := sensitiveDumpAllValues(sensitiveDumpConfigDir); err != nil {
			l.WithError(err).Warn("Dumping the config was not possible.")
		}
	}

	AddWatcher(func(event fsnotify.Event) error {
		if l.LeakSensitiveData() && sensitiveDumpConfigDir != "" {
			if err := sensitiveDumpAllValues(sensitiveDumpConfigDir); err != nil {
				l.WithError(err).Warn("Dumping the config was not possible.")
			}
		}
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
