package viperx

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"

	"github.com/ory/viper"

	"github.com/ory/x/jsonschemax"
)

// BindEnvsToSchema uses all keys it can find from ``
func BindEnvsToSchema(schema json.RawMessage) error {
	keys, err := jsonschemax.ListPathsBytes(schema)
	if err != nil {
		return err
	}

	viper.AutomaticEnv()
	viper.SetTypeByDefaultValue(true)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	for _, key := range keys {
		if err := viper.BindEnv(key.Name); err != nil {
			return errors.WithStack(err)
		}

		if key.Default != nil {
			viper.SetDefault(key.Name, key.Default)
		} else {
			if key.Type != "" {
				viper.SetDefault(key.Name, key.Type)
			}
		}
	}

	return nil
}
