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
	_, defaults, err := getSchemaKeys(string(schema), string(schema), []string{}, []string{})
	if err != nil {
		return err
	}

	for key, def := range defaults {
		viper.SetDefault(key, def)
	}

	return nil
}

func getSchemaKeys(root, current string, parents []string, traversed []string) ([]string, map[string]interface{}, error) {
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
		return nil, nil, nil
	}

	defaults := map[string]interface{}{}
	var paths []string
	var err error

	traversed = append(traversed, keys[foundKey])
	switch foundKey {
	case properties:
		result.ForEach(func(k, v gjson.Result) bool {
			this := append(parents, k.String())
			paths = append(paths, strings.Join(this, "."))
			joined := strings.Join(this, ".")

			if d := v.Get("default"); d.Exists() {
				defaults[strings.Join(this, ".")] = d.Value()
			} else if t := v.Get("type"); t.Exists() {
				switch t.String() {
				case "array":
					defaults[joined] = []interface{}{}
				case "boolean":
					defaults[joined] = false
				case "string":
					defaults[joined] = ""
				case "number":
					defaults[joined] = 0
				case "object":
					defaults[joined] = map[string]interface{}{}
				}
			}

			if v.IsObject() {
				merge, def, innerErr := getSchemaKeys(root, v.Raw, this, traversed)
				if innerErr != nil {
					err = innerErr
					return false // break out
				}
				for k, v := range def {
					defaults[k] = v
				}
				paths = append(paths, merge...)
			}
			return true // run through all keys
		})
	case ref:
		defpath := result.String()
		if !strings.HasPrefix(defpath, "#/definitions/") {
			return nil, nil, errors.New("only references to #/definitions/ are supported")
		}
		path := strings.ReplaceAll(strings.TrimPrefix(defpath, "#/"), "/", ".")
		if stringslice.HasI(traversed, path) {
			return nil, nil, errors.Errorf("detected circular dependency in schema path: %v", traversed)
		}
		merge, def, err := getSchemaKeys(root, gjson.Get(root, path).Raw, parents, append(traversed, path))
		if err != nil {
			return nil, nil, err
		}
		for k, v := range def {
			defaults[k] = v
		}
		paths = append(paths, merge...)
	case allOf:
		fallthrough
	case oneOf:
		fallthrough
	case anyOf:
		for _, item := range result.Array() {
			merge, def, err := getSchemaKeys(root, item.Raw, parents, traversed)
			if err != nil {
				return nil, nil, err
			}
			for k, v := range def {
				defaults[k] = v
			}
			paths = append(paths, merge...)
		}
	default:
		panic(fmt.Sprintf("found unexpected key: %d", foundKey))
	}

	if err != nil {
		return nil, nil, err
	}

	return stringslice.Unique(paths), defaults, err
}
