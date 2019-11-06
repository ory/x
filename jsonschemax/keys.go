package jsonschemax

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/tidwall/gjson"

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

type Path struct {
	Name    string
	Default interface{}
	Type    interface{}
}

func ListPaths(root string) ([]Path, error) {
	return listPaths(root, root, nil, nil)
}

func ListPathsBytes(root []byte) ([]Path, error) {
	return listPaths(string(root), string(root), nil, nil)
}

func parsePrimitiveType(current gjson.Result, path string) (interface{}, error) {
	if t := current.Get("const"); t.Exists() {
		switch t.Type {
		case gjson.String:
			return "", nil
		case gjson.Null:
			return nil, nil
		case gjson.Number:
			return float64(0), nil
		case gjson.JSON:
			return "", nil
		case gjson.True:
			return true, nil
		case gjson.False:
			return false, nil
		}

		panic(fmt.Sprintf(`unexpected gjson.Type result %v in path: %s.const`, t.Type, path))
	}

	t := current.Get("type")
	if !t.Exists() {
		return nil, errors.Errorf(`neither "type" nor "const" were defined in JSON Schema for path: %s`, path)
	}

	switch t.String() {
	case "string":
		return "", nil
	case "number":
		return float64(0), nil
	case "boolean":
		return false, nil
	case "null":
		return nil, nil
	case "object":
		return map[string]interface{}{}, nil
	case "array":
		if ic := current.Get("items.const"); ic.Exists() {
			switch ic.Type {
			case gjson.String:
				return []string{}, nil
			case gjson.Null:
				return []interface{}{}, nil
			case gjson.Number:
				return []float64{}, nil
			case gjson.JSON:
				return []string{}, nil
			case gjson.True:
				fallthrough
			case gjson.False:
				return []bool{}, nil
			}

			panic(fmt.Sprintf(`unexpected gjson.Type result %v in path: %s.items.const`, t.Type, path))
		}

		it := current.Get("items.type")
		if !it.Exists() {
			return nil, errors.Errorf(`neither "type" nor "const" were defined in JSON Schema for path: %s.items`, path)
		}

		switch it.String() {
		case "string":
			return []string{}, nil
		case "number":
			return []float64{}, nil
		case "boolean":
			return []bool{}, nil
		case "object":
			return []map[string]interface{}{}, nil
		case "array":
			fallthrough
		case "null":
			return []interface{}{}, nil
			// case "array":
			// 	return nil, errors.Errorf("nested JSON Schema arrays can currently not be decomposed in path: %s.items", path)
		}
	}

	return nil, errors.Errorf(`unexpected type "%+v" in JSON Schema path: %s`, t, path)
}

func listPaths(root, current string, parents []string, traversed []string) ([]Path, error) {
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

	// defaults := map[string]interface{}{}

	// None of "properties, $ref, allOf, anyOf, oneOf" were found.
	if foundKey == none {
		var (
			keyPath    = strings.Join(parents, ".")
			keyDefault interface{}
		)

		if def := gjson.Get(current, "default"); def.Exists() {
			keyDefault = def.Value()
		}

		keyType, err := parsePrimitiveType(gjson.Parse(current), keyPath)
		if err != nil {
			return nil, err
		}

		return []Path{{Name: keyPath, Default: keyDefault, Type: keyType}}, nil
	}

	var paths []Path
	var err error
	traversed = append(traversed, keys[foundKey])
	switch foundKey {
	case properties:
		result.ForEach(func(k, v gjson.Result) bool {
			this := append(parents, k.String())
			if !v.IsObject() {
				err = errors.Errorf(`path "%s" must be an object but got: %s`, strings.Join(this, "."), v.Raw)
				return false
			}

			// paths = append(paths, Path{Name: strings.Join(this, ".")})
			merge, innerErr := listPaths(root, v.Raw, this, traversed)
			if innerErr != nil {
				err = innerErr
				return false // break out
			}

			paths = append(paths, merge...)
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
		merge, err := listPaths(root, gjson.Get(root, path).Raw, parents, append(traversed, path))
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
			merge, err := listPaths(root, item.Raw, parents, traversed)
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

	return paths, err
}
