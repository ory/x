package viperx

import (
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/ory/viper"

	"github.com/ory/x/logrusx"
)

var cfgFile string
var watchers []func(event fsnotify.Event)
var watcherLock sync.Mutex

// RegisterConfigFlag registers the --config / -c flag.
func RegisterConfigFlag(c *cobra.Command, applicationName string) {
	c.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", `Path to config file. Supports .json, .yaml, .yml, .toml. Default is "$HOME/.`+applicationName+`.(yaml|yml|toml|json)"`)
}

// WatchOptions configures WatchConfig.
type WatchOptions struct {
	// Immutables are keys that cause OnImmutableChange to be fired when modified.
	Immutables []string
	// OnImmutableChange - see Immutables.
	OnImmutableChange func(immutable string)
}

func AddWatcher(f func(event fsnotify.Event)) {
	watcherLock.Lock()
	defer watcherLock.Unlock()
	watchers = append(watchers, f)
}

// WatchConfig is a helper makes watching configuration files easy.
func WatchConfig(l logrus.FieldLogger, o *WatchOptions) {
	if l == nil {
		l = logrusx.New()
	}

	if o == nil {
		o = new(WatchOptions)
	}

	for _, key := range o.Immutables {
		// This ensures that the keys are all set
		viper.Get(key)
	}

	watcherLock.Lock()
	watchers = nil
	watcherLock.Unlock()

	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		l.WithField("file", in.Name).WithField("operator", in.Op.String()).Debug("The configuration has changed and was reloaded.")

		if o.OnImmutableChange != nil {
			for _, key := range o.Immutables {
				if viper.HasChanged(key) {
					o.OnImmutableChange(key)
				}
			}
		}

		watcherLock.Lock()
		for _, w := range watchers {
			w(in)
		}
		watcherLock.Unlock()
	})
}

// InitializeConfig initializes viper.
func InitializeConfig(applicationName string, homeOverride string, l logrus.FieldLogger) {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			logrusx.New().WithField("error", err.Error()).Fatal("Unable to locate home directory")
		}

		if homeOverride != "" {
			home = homeOverride
		}

		// Search config in home directory with the application name and a dot prepended (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName("." + applicationName)
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv() // read in environment variables that match

	err := viper.ReadInConfig()
	if l == nil {
		l = logrusx.New()
	}

	if err == nil {
		l.WithField("path", viper.ConfigFileUsed()).Info("Config file loaded successfully.")
	} else {
		switch t := err.(type) {
		case viper.UnsupportedConfigError:
			if len(t) == 0 {
				l.WithError(err).Warn("No config file was defined and no file was found in $HOME/." + applicationName + ".yaml")
			} else {
				l.WithError(err).WithField("extension", t).Fatal("Unsupported configuration type")
			}
		case *viper.ConfigFileNotFoundError:
			l.WithError(err).Warn("No config file was defined and no file was found in $HOME/." + applicationName + ".yaml")
		case viper.ConfigFileNotFoundError:
			l.WithError(err).Warn("No config file was defined and no file was found in $HOME/." + applicationName + ".yaml")
		default:
			l.
				WithField("path", viper.ConfigFileUsed()).
				WithError(err).
				Fatal("Unable to open config file. Make sure it exists and the process has sufficient permissions to read it")
		}
		return
	}
}
