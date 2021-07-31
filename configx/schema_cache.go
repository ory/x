package configx

import (
	"crypto/sha256"
	"fmt"

	"github.com/dgraph-io/ristretto"

	"github.com/ory/jsonschema/v3"
)

var schemaCacheConfig = &ristretto.Config{
	// Hold up to 25 schemas in cache. Usually we only need one.
	MaxCost:            25,
	NumCounters:        250,
	BufferItems:        64,
	Metrics:            false,
	IgnoreInternalCost: true,
}
var schemaCache, _ = ristretto.NewCache(schemaCacheConfig)

func getSchema(schema []byte) (*jsonschema.Schema, error) {
	key := fmt.Sprintf("%x", sha256.Sum256(schema))
	if val, found := schemaCache.Get(key); found {
		if validator, ok := val.(*jsonschema.Schema); ok {
			return validator, nil
		}
		schemaCache.Del(key)
	}

	schemaID, comp, err := newCompiler(schema)
	if err != nil {
		return nil, err
	}

	validator, err := comp.Compile(schemaID)
	if err != nil {
		return nil, err
	}

	schemaCache.Set(key, validator, 1)
	schemaCache.Wait()
	return validator, nil
}
