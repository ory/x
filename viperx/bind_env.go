package viperx

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/tidwall/gjson"

	"github.com/ory/viper"

	"github.com/ory/x/stringslice"
)

const (
	none = iota - 1
	properties
	ref
	allOf
	anyOf
	oneOf
)

var keys = []string{
	"properties",
	"$ref",
	"allOf",
	"anyOf",
	"oneOf",
}

// BindEnvsToSchema uses all keys it can find from ``
func BindEnvsToSchema(schema json.RawMessage) error {
	keys, err := getSchemaKeys(string(schema), string(schema), []string{}, []string{})
	if err != nil {
		return err
	}

	for _, key := range keys {
		if err := viper.BindEnv(key); err != nil {
			return err
		}
	}

	return nil
}

func getSchemaKeys(root, current string, parents []string, traversed []string) ([]string, error) {
	var foundKey = -1
	var result gjson.Result
	for i, value := range gjson.GetMany(
		current,
		keys...,
	) {
		if value.Exists() {
			foundKey = i
			result = value
			break
		}
	}

	if foundKey == none {
		return nil, nil
	}

	var paths []string
	var err error

	traversed = append(traversed, keys[foundKey])
	switch foundKey {
	case properties:
		result.ForEach(func(k, v gjson.Result) bool {
			this := append(parents, k.String())
			paths = append(paths, strings.Join(this, "."))
			if v.IsObject() {
				merge, innerErr := getSchemaKeys(root, v.Raw, this, traversed)
				if innerErr != nil {
					err = innerErr
					return false // break out
				}
				paths = append(paths, merge...)
			}
			return true // run through all keys
		})
	case ref:
		defpath := result.String()
		if !strings.HasPrefix(defpath, "#/definitions/") {
			return nil, errors.New("only references to #/definitions/ are supported")
		}
		path := strings.ReplaceAll(strings.TrimPrefix(defpath, "#/"), "/", ".")
		if stringslice.HasI(traversed, path) {
			return nil, errors.Errorf("detected circular dependency in schema path: %v", traversed)
		}
		merge, err := getSchemaKeys(root, gjson.Get(root, path).Raw, parents, append(traversed, path))
		if err != nil {
			return nil, err
		}
		paths = append(paths, merge...)
	case allOf:
		fallthrough
	case oneOf:
		fallthrough
	case anyOf:
		for _, item := range result.Array() {
			merge, err := getSchemaKeys(root, item.Raw, parents, traversed)
			if err != nil {
				return nil, err
			}
			paths = append(paths, merge...)
		}
	default:
		panic(fmt.Sprintf("found unexpected key: %d", foundKey))
	}

	if err != nil {
		return nil, err
	}

	return stringslice.Unique(paths), err
}
