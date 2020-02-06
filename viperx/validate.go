package viperx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/ory/jsonschema/v3"
	"github.com/ory/viper"

	"github.com/ory/x/errorsx"
	"github.com/ory/x/jsonschemax"
)

// ValidateFromURL validates the viper config by loading the schema from a URL
//
// Uses Validate internally.
func ValidateFromURL(url string) error {
	buf, err := jsonschema.LoadURL(url)
	if err != nil {
		return errors.WithStack(err)
	}

	result, err := ioutil.ReadAll(buf)
	if err != nil {
		return errors.WithStack(err)
	}

	return Validate(url, result)
}

// Validate validates the viper config
//
// If env vars are supported, they must be bound using viper.BindEnv.
func Validate(name string, content []byte) error {
	if err := BindEnvsToSchema(content); err != nil {
		return errors.WithStack(err)
	}

	viper.SetTypeByDefaultValue(true)

	c := jsonschema.NewCompiler()
	if err := c.AddResource(name, bytes.NewBuffer(content)); err != nil {
		return errors.WithStack(err)
	}

	s, err := c.Compile(name)
	if err != nil {
		return errors.WithStack(err)
	}

	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(viper.AllSettings()); err != nil {
		return errors.WithStack(err)
	}

	if err := s.Validate(&b); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// LoggerWithValidationErrorFields adds all validation errors as fields to the logger.
func LoggerWithValidationErrorFields(l logrus.FieldLogger, err error) logrus.FieldLogger {
	entry := l

	switch e := errorsx.Cause(err).(type) {
	case *jsonschema.ValidationError:
		pointer, message := jsonschemaFormatError(e)
		entry = l.
			WithField("config_file", viper.ConfigFileUsed()).
			WithField("config_key", pointer).
			WithField("validation_error", message)

		for idx, cause := range e.Causes {
			pointer, message := jsonschemaFormatError(cause)
			entry = l.
				WithField(fmt.Sprintf("config_key[%d]", idx), pointer).
				WithField(fmt.Sprintf("validation_error[%d]", idx), message)
		}
	}

	return entry
}

func jsonschemaFormatError(e *jsonschema.ValidationError) (pointer, message string) {
	var err error

	pointer = e.InstancePtr
	message = e.Message
	switch ctx := e.Context.(type) {
	case *jsonschema.ValidationErrorContextRequired:
		if len(ctx.Missing) > 0 {
			message = "one or more required properties are missing"
			pointer = ctx.Missing[0]
		}
	}

	// We can ignore the error as it will simply echo the pointer.
	pointer, err = jsonschemax.JSONPointerToDotNotation(pointer)
	if err != nil {
		pointer = e.InstancePtr
	}

	return
}
