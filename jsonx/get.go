package jsonx

import (
	"reflect"
	"strings"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// AllValidJSONKeys returns all JSON keys from the struct or *struct type.
// It does not return keys from nested structs, but embedded structs.
func AllValidJSONKeys(s interface{}) (keys []string) {
	t := reflect.TypeOf(s)
	v := reflect.ValueOf(s)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if jsonTag := f.Tag.Get("json"); jsonTag != "" {
			if jsonTag == "-" {
				continue
			}
			keys = append(keys, strings.Split(jsonTag, ",")[0])
		} else if f.IsExported() {
			if f.Anonymous {
				keys = append(keys, AllValidJSONKeys(v.Field(i).Interface())...)
			} else {
				keys = append(keys, f.Name)
			}
		}
	}
	return keys
}

// ParseEnsureKeys returns a result that has the GetRequireValidKey function.
func ParseEnsureKeys(original interface{}, raw []byte) *result {
	return &result{
		keys:   AllValidJSONKeys(original),
		result: gjson.ParseBytes(raw),
	}
}

type result struct {
	result gjson.Result
	keys   []string
}

// GetRequireValidKey ensures that the key is valid before returning the result.
func (r *result) GetRequireValidKey(t require.TestingT, key string) gjson.Result {
	require.Contains(t, r.keys, key)
	return r.result.Get(key)
}

func GetRequireValidKey(t require.TestingT, original interface{}, raw []byte, key string) gjson.Result {
	return ParseEnsureKeys(original, raw).GetRequireValidKey(t, key)
}
