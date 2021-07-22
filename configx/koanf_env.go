package configx

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"unsafe"

	"github.com/dgraph-io/ristretto"
	"github.com/knadh/koanf/providers/env"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"

	"github.com/ory/x/castx"
	"github.com/ory/x/jsonschemax"
)

var pathCache, _ = ristretto.NewCache(&ristretto.Config{
	NumCounters: 1000,
	MaxCost:     100000000, // maximum cost of cache (100MB).
	BufferItems: 64,        // number of keys per Get buffer.
})

func NewKoanfEnv(prefix string, schema []byte) (*env.Env, error) {
	cacheKey := fmt.Sprintf("%x", sha256.Sum256(schema))

	pathsFromCache, found := pathCache.Get(cacheKey)
	paths, ok := pathsFromCache.([]jsonschemax.Path)
	if !found || !ok {
		id, compiler, err := newCompiler(schema)
		if err != nil {
			return nil, err
		}

		paths, err = jsonschemax.ListPaths(id, compiler)
		if err != nil {
			return nil, err
		}

		_ = pathCache.Set(cacheKey, paths, int64(unsafe.Sizeof(paths)))
	}

	decode := func(value string) (v interface{}) {
		b := []byte(value)
		var arr []interface{}
		if err := json.Unmarshal(b, &arr); err == nil {
			return &arr
		}
		h := map[string]interface{}{}
		if err := json.Unmarshal(b, &h); err == nil {
			return &h
		}
		return nil
	}

	return env.ProviderWithValue(prefix, ".", func(key string, value string) (string, interface{}) {
		key = strings.Replace(strings.ToLower(strings.TrimPrefix(key, prefix)), "_", ".", -1)
		for _, path := range paths {
			normalized := strings.Replace(path.Name, "_", ".", -1)

			if normalized == key {
				switch path.TypeHint {
				case jsonschemax.String:
					return path.Name, cast.ToString(value)
				case jsonschemax.Float:
					return path.Name, cast.ToFloat64(value)
				case jsonschemax.Int:
					return path.Name, cast.ToInt64(value)
				case jsonschemax.Bool:
					return path.Name, cast.ToBool(value)
				case jsonschemax.Nil:
					return path.Name, nil
				case jsonschemax.BoolSlice:
					if !gjson.Valid(value) {
						return path.Name, cast.ToBoolSlice(value)
					}
					fallthrough
				case jsonschemax.StringSlice:
					if !gjson.Valid(value) {
						return path.Name, castx.ToStringSlice(value)
					}
					fallthrough
				case jsonschemax.IntSlice:
					if !gjson.Valid(value) {
						return path.Name, cast.ToIntSlice(value)
					}
					fallthrough
				case jsonschemax.FloatSlice:
					if !gjson.Valid(value) {
						return path.Name, castx.ToFloatSlice(value)
					}
					fallthrough
				case jsonschemax.JSON:
					return path.Name, decode(value)
				default:
					return path.Name, value
				}
			}
		}

		return "", nil
	}), nil
}
