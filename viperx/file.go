package viperx

import (
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ory/x/logrusx"
)

var cfgFile string

func RegisterConfigFlag(c *cobra.Command, applicationName string) {
	c.PersistentFlags().StringVarP(&cfgFile, "config", "p", "", "Path to config file. Supports .json, .yaml, .yml, .toml. Default is $HOME/."+applicationName+".(yaml|yml|toml|json)")
}

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

	viper.SetDefault("LOG_LEVEL", "info")
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
	}
}
