package configx

import (
	"github.com/knadh/koanf/maps"
	"github.com/pkg/errors"

	"github.com/ory/jsonschema/v3"
	"github.com/ory/x/jsonschemax"
)

type KoanfSchemaDefaults struct {
	c   *jsonschema.Compiler
	uri string
}

func NewKoanfSchemaDefaults(schema []byte) (*KoanfSchemaDefaults, error) {
	id, c, err := newCompiler(schema)
	if err != nil {
		return nil, err
	}
	return &KoanfSchemaDefaults{c: c, uri: id}, nil
}

func (k *KoanfSchemaDefaults) ReadBytes() ([]byte, error) {
	return nil, errors.New("schema defaults provider does not support this method")
}

func (k *KoanfSchemaDefaults) Read() (map[string]interface{}, error) {
	keys, err := jsonschemax.ListPaths(k.uri, k.c)
	if err != nil {
		return nil, err
	}

	values := map[string]interface{}{}
	for _, key := range keys {
		if key.Default != nil {
			values[key.Name] = key.Default
		}
	}

	return maps.Unflatten(values, "."), nil
}
