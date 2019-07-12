package viperx

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/ory/gojsonschema"
)

type ValidationErrors []gojsonschema.ResultError

func (err ValidationErrors) Error() string {
	return fmt.Sprintf("%s", err[0])
}

// Validate validates the viper config. If env vars are supported, they must be bound using viper.BindEnv
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
