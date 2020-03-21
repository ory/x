package viperx

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cast"

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

		viper.SetType(key.Name, key.Type)
		if key.Default != nil {
			// key.Default will be of type []interface{} whenever it is an array in the schema
			switch key.Type.(type) {
			case []string:
				def, err := cast.ToStringSliceE(key.Default)
				if err != nil {
					return errors.WithStack(err)
				}
				viper.SetDefault(key.Name, def)
			case []float64:
				switch def := key.Default.(type) {
				case []interface{}:
					var r []float64
					for _, i := range def {
						// we first cast to string as json.Number is the type of numbers
						s, err := cast.ToStringE(i)
						if err != nil {
							return errors.WithStack(err)
						}
						f, err := cast.ToFloat64E(s)
						if err != nil {
							return errors.WithStack(err)
						}
						r = append(r, f)
					}
					viper.SetDefault(key.Name, r)
				default:
					viper.SetDefault(key.Name, key.Default)
				}
			default:
				viper.SetDefault(key.Name, key.Default)
			}
		}
	}

	return nil
}
