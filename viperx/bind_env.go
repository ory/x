package viperx

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/ory/jsonschema/v3"

	"github.com/ory/viper"

	"github.com/ory/x/jsonschemax"
)

// BindEnvsToSchema uses all keys it can find from ``
func BindEnvsToSchema(schema json.RawMessage) error {
	compiler := jsonschema.NewCompiler()
	id := fmt.Sprintf("%x.json", sha256.Sum256(schema))
	if err := compiler.AddResource(id, bytes.NewReader(schema)); err != nil {
		return errors.WithStack(err)
	}
	compiler.ExtractAnnotations = true

	keys, err := jsonschemax.ListPaths(id, compiler)
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
