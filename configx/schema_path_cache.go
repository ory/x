// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package configx

import (
	"crypto/sha256"
	"fmt"

	"github.com/ory/x/jsonschemax"

	"github.com/dgraph-io/ristretto"

	"github.com/ory/jsonschema/v3"
)

var schemaPathCacheConfig = &ristretto.Config{
	// Hold up to 25 schemas in cache. Usually we only need one.
	MaxCost:            250,
	NumCounters:        2500,
	BufferItems:        64,
	Metrics:            false,
	IgnoreInternalCost: true,
}

var schemaPathCache, _ = ristretto.NewCache(schemaPathCacheConfig)

func getSchemaPaths(rawSchema []byte, schema *jsonschema.Schema) ([]jsonschemax.Path, error) {
	key := fmt.Sprintf("%x", sha256.Sum256(rawSchema))
	if val, found := schemaPathCache.Get(key); found {
		if validator, ok := val.([]jsonschemax.Path); ok {
			return validator, nil
		}
		schemaPathCache.Del(key)
	}

	keys, err := jsonschemax.ListPathsWithInitializedSchemaAndArraysIncluded(schema)
	if err != nil {
		return nil, err
	}

	schemaPathCache.Set(key, keys, 1)
	schemaPathCache.Wait()
	return keys, nil
}
