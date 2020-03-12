package viperx

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
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

		if key.Default == nil {
			// we have to set a default this way so that viper can use it's type
			viper.SetDefault(key.Name, key.Type)
			continue
		}

		if fmt.Sprintf("%T", key.Type) == fmt.Sprintf("%T", key.Default) {
			// key.Type and key.Default are of the same type, no conversion needed
			viper.SetDefault(key.Name, key.Default)
			continue
		}

		// this type conversion has to be improved
		switch key.Type.(type) {
		case []string:
			if reflect.TypeOf(key.Default).Kind() == reflect.Slice {
				var d []string
				v := reflect.ValueOf(key.Default)

				for i := 0; i < v.Len(); i++ {
					d = append(d, fmt.Sprintf("%v", v.Index(i)))
				}

				viper.SetDefault(key.Name, d)
				continue
			}
		}
	}

	return nil
}
