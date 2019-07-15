package viperx

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/ory/viper"

	"github.com/ory/gojsonschema"
)

// ValidationErrors is a wrapper for []gojsonschema.ResultError that implements the error interface.
type ValidationErrors []gojsonschema.ResultError

// Error returns a string representation of the JSON Schema Validation Errors.
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

	res, err := s.Validate(gojsonschema.NewGoLoader(viper.AllSettings()))
	if err != nil {
		return errors.WithStack(err)
	}

	if !res.Valid() {
		return errors.WithStack(ValidationErrors(res.Errors()))
	}

	return nil
}
