package viperx

import (
	"bytes"
	"encoding/json"
	"io/ioutil"

	"github.com/pkg/errors"

	"github.com/ory/jsonschema/v3"
	"github.com/ory/viper"
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
