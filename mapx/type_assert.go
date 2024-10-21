// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package mapx

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"
)

// ErrKeyDoesNotExist is returned when the key does not exist in the map.
var ErrKeyDoesNotExist = errors.New("key is not present in map")

// ErrKeyCanNotBeTypeAsserted is returned when the key can not be type asserted.
var ErrKeyCanNotBeTypeAsserted = errors.New("key could not be type asserted")

// GetString returns a string for a given key in values.
func GetString[K comparable](values map[K]any, key K) (string, error) {
	if v, ok := values[key]; !ok {
		return "", ErrKeyDoesNotExist
	} else if sv, ok := v.(string); !ok {
		return "", ErrKeyCanNotBeTypeAsserted
	} else {
		return sv, nil
	}
}

// GetStringSlice returns a string slice for a given key in values.
func GetStringSlice[K comparable](values map[K]any, key K) ([]string, error) {
	if v, ok := values[key]; !ok {
		return []string{}, ErrKeyDoesNotExist
	} else if sv, ok := v.([]string); ok {
		return sv, nil
	} else if sv, ok := v.([]any); ok {
		vs := make([]string, len(sv))
		for k, v := range sv {
			vv, ok := v.(string)
			if !ok {
				return []string{}, ErrKeyCanNotBeTypeAsserted
			}
			vs[k] = vv
		}
		return vs, nil
	}
	return []string{}, ErrKeyCanNotBeTypeAsserted
}

// GetTime returns a string slice for a given key in values.
func GetTime[K comparable](values map[K]any, key K) (time.Time, error) {
	v, ok := values[key]
	if !ok {
		return time.Time{}, ErrKeyDoesNotExist
	}

	if sv, ok := v.(time.Time); ok {
		return sv, nil
	} else if sv, ok := v.(int64); ok {
		return time.Unix(sv, 0), nil
	} else if sv, ok := v.(int32); ok {
		return time.Unix(int64(sv), 0), nil
	} else if sv, ok := v.(int); ok {
		return time.Unix(int64(sv), 0), nil
	} else if sv, ok := v.(float64); ok {
		return time.Unix(int64(sv), 0), nil
	} else if sv, ok := v.(float32); ok {
		return time.Unix(int64(sv), 0), nil
	}

	return time.Time{}, ErrKeyCanNotBeTypeAsserted
}

// GetInt64Default returns a int64 or the default value for a given key in values.
func GetInt64Default[K comparable](values map[K]any, key K, defaultValue int64) int64 {
	f, err := GetInt64(values, key)
	if err != nil {
		return defaultValue
	}
	return f
}

// GetInt64 returns an int64 for a given key in values.
func GetInt64[K comparable](values map[K]any, key K) (int64, error) {
	v, ok := values[key]
	if !ok {
		return 0, ErrKeyDoesNotExist
	}
	switch v := v.(type) {
	case json.Number:
		return v.Int64()
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case uint:
		if v > math.MaxInt64 {
			return 0, errors.New("value is out of range")
		}
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		if v > math.MaxInt64 {
			return 0, errors.New("value is out of range")
		}
		return int64(v), nil
	}
	return 0, ErrKeyCanNotBeTypeAsserted
}

// GetInt32Default returns a int32 or the default value for a given key in values.
func GetInt32Default[K comparable](values map[K]any, key K, defaultValue int32) int32 {
	f, err := GetInt32(values, key)
	if err != nil {
		return defaultValue
	}
	return f
}

// GetInt32 returns an int32 for a given key in values.
func GetInt32[K comparable](values map[K]any, key K) (int32, error) {
	v, err := GetInt64(values, key)
	if err != nil {
		return 0, err
	}
	if v > math.MaxInt32 || v < math.MinInt32 {
		return 0, errors.New("value is out of range")
	}
	return int32(v), nil
}

// GetIntDefault returns a int or the default value for a given key in values.
func GetIntDefault[K comparable](values map[K]any, key K, defaultValue int) int {
	f, err := GetInt(values, key)
	if err != nil {
		return defaultValue
	}
	return f
}

// GetInt returns an int for a given key in values.
func GetInt[K comparable](values map[K]any, key K) (int, error) {
	v, err := GetInt64(values, key)
	if err != nil {
		return 0, err
	}
	if v > math.MaxInt || v < math.MinInt {
		return 0, errors.New("value is out of range")
	}
	return int(v), nil
}

// GetFloat32Default returns a float32 or the default value for a given key in values.
func GetFloat32Default[K comparable](values map[K]any, key K, defaultValue float32) float32 {
	f, err := GetFloat32(values, key)
	if err != nil {
		return defaultValue
	}
	return f
}

// GetFloat32 returns a float32 for a given key in values.
func GetFloat32[K comparable](values map[K]any, key K) (float32, error) {
	if v, ok := values[key]; !ok {
		return 0, ErrKeyDoesNotExist
	} else if j, ok := v.(json.Number); ok {
		v, err := j.Float64()
		return float32(v), err
	} else if sv, ok := v.(float32); ok {
		return sv, nil
	}
	return 0, ErrKeyCanNotBeTypeAsserted
}

// GetFloat64Default returns a float64 or the default value for a given key in values.
func GetFloat64Default[K comparable](values map[K]any, key K, defaultValue float64) float64 {
	f, err := GetFloat64(values, key)
	if err != nil {
		return defaultValue
	}
	return f
}

// GetFloat64 returns a float64 for a given key in values.
func GetFloat64[K comparable](values map[K]any, key K) (float64, error) {
	if v, ok := values[key]; !ok {
		return 0, ErrKeyDoesNotExist
	} else if j, ok := v.(json.Number); ok {
		return j.Float64()
	} else if sv, ok := v.(float64); ok {
		return sv, nil
	}
	return 0, ErrKeyCanNotBeTypeAsserted
}

// GetStringDefault returns a string or the default value for a given key in values.
func GetStringDefault[K comparable](values map[K]any, key K, defaultValue string) string {
	if s, err := GetString(values, key); err == nil {
		return s
	}
	return defaultValue
}

// GetStringSliceDefault returns a string slice or the default value for a given key in values.
func GetStringSliceDefault[K comparable](values map[K]any, key K, defaultValue []string) []string {
	if s, err := GetStringSlice(values, key); err == nil {
		return s
	}
	return defaultValue
}

// KeyStringToInterface converts map[string]any to map[any]any
// Deprecated: with generics, this should not be necessary anymore.
func KeyStringToInterface(i map[string]any) map[any]any {
	o := make(map[any]any)
	for k, v := range i {
		o[k] = v
	}
	return o
}

// ToJSONMap converts all map[any]any occurrences (nested as well) to map[string]any.
// Deprecated: with generics, this should not be necessary anymore.
func ToJSONMap(i any) any {
	switch t := i.(type) {
	case []any:
		for k, v := range t {
			t[k] = ToJSONMap(v)
		}
		return t
	case map[string]any:
		for k, v := range t {
			t[k] = ToJSONMap(v)
		}
		return t
	case map[any]any:
		res := make(map[string]any)
		for k, v := range t {
			res[fmt.Sprintf("%s", k)] = ToJSONMap(v)
		}
		return res
	}

	return i
}
