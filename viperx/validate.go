package viperx

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/ory/gojsonschema"
)

// ValidationErrors is a wrapper for []gojsonschema.ResultError that implements the error interface.
type ValidationErrors []gojsonschema.ResultError

func (err ValidationErrors) Error() string {
	errs := make([]string, len(err))
	for k, v := range err {
		errs[k] = fmt.Sprintf("%s", v)
	}
	return fmt.Sprintf("%+v", errs)
}

// Validate validates the viper config. If env vars are supported, they must be bound using viper.BindEnv.
func Validate(schema gojsonschema.JSONLoader) error {
	s, err := gojsonschema.NewSchema(schema)
	if err != nil {
		return errors.WithStack(err)
	}

	res, err := s.Validate(gojsonschema.NewGoLoader(toMapStringInterface(viper.AllSettings())))
	if err != nil {
		return errors.WithStack(err)
	}

	if !res.Valid() {
		return errors.WithStack(ValidationErrors(res.Errors()))
	}

	return nil
}

// toMapStringInterface is a workaround for https://github.com/spf13/viper/issues/730
// and https://github.com/go-yaml/yaml/issues/139
func toMapStringInterface(in interface{}) interface{} {
	switch t := in.(type) {
	case map[string]interface{}:
		for k, v := range t {
			t[k] = toMapStringInterface(v)
		}
		return t
	case map[interface{}]interface{}:
		nt := make(map[string]interface{})
		for k, v := range t {
			nt[fmt.Sprintf("%s", k)] = toMapStringInterface(v)
		}
		return nt
	case []interface{}:
		for k, v := range t {
			t[k] = toMapStringInterface(v)
		}
		return t
	default:
		return in
	}
}
