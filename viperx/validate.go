package viperx

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/ory/viper"

	"github.com/ory/gojsonschema"
)

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
		return errors.WithStack(res.Errors())
	}

	return nil
}

// LoggerWithValidationErrorFields adds all validation errors as fields to the logger.
func LoggerWithValidationErrorFields(l logrus.FieldLogger, err error) logrus.FieldLogger {
	if errs, ok := errors.Cause(err).(gojsonschema.ResultErrors); ok {
		for k, err := range errs {
			l = l.WithField(fmt.Sprintf("validation_error[%d]", k), fmt.Sprintf("%+v", err))
		}
	}

	return l
}
