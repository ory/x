package jsonschemax

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/santhosh-tekuri/jsonschema/v2"
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

type byName []Path

func (s byName) Len() int           { return len(s) }
func (s byName) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s byName) Less(i, j int) bool { return s[i].Name < s[j].Name }

// Path represents a JSON Schema Path.
type Path struct {
	// Name is the JSON path name.
	Name string

	// Default is the default value of that path.
	Default interface{}

	// Type is a prototype (e.g. float64(0)) of the path type.
	Type interface{}
}

// ListPaths lists all paths of a JSON Schema. Will return an error
// if circular dependencies are found.
func ListPaths(ref string, compiler *jsonschema.Compiler) ([]Path, error) {
	if compiler == nil {
		compiler = jsonschema.NewCompiler()
	}

	compiler.ExtractAnnotations = true
	pointers := map[string]bool{}

	schema, err := compiler.Compile(ref)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	paths, err := listPaths(schema, nil, pointers)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	sort.Sort(paths)
	return makeUnique(paths)
}

func makeUnique(in byName) (byName, error) {
	cache := make(map[string]Path)
	for _, p := range in {
		vc, ok := cache[p.Name]
		if !ok {
			cache[p.Name] = p
			continue
		}

		if fmt.Sprintf("%T", p.Type) != fmt.Sprintf("%T", p.Type) {
			return nil, errors.Errorf("multiple types %+v are not supported for path: %s", []interface{}{p.Type, vc.Type}, p.Name)
		}

		if vc.Default == nil {
			cache[p.Name] = p
		}
	}

	k := 0
	out := make([]Path, len(cache))
	for _, v := range cache {
		out[k] = v
		k++
	}

	paths := byName(out)
	sort.Sort(paths)
	return paths, nil
}

func appendPointer(in map[string]bool, pointer *jsonschema.Schema) map[string]bool {
	out := make(map[string]bool)
	for k, v := range in {
		out[k] = v
	}
	out[fmt.Sprintf("%p", pointer)] = true
	return out
}

func listPaths(schema *jsonschema.Schema, parents []string, pointers map[string]bool) (byName, error) {
	if _, ok := pointers[fmt.Sprintf("%p", schema)]; ok {
		return nil, errors.Errorf("detected circular dependency in schema path: %s", strings.Join(parents, "."))
	}

	var pathType interface{}
	var paths []Path
	if len(schema.Constant) > 0 {
		switch schema.Constant[0].(type) {
		case float64, int64, json.Number:
			pathType = float64(0)
		case string:
			pathType = ""
		case bool:
			pathType = false
		default:
			pathType = schema.Constant[0]
		}
	} else if len(schema.Types) > 2 {
		pathType = nil
	} else if len(schema.Types) > 0 {
		switch schema.Types[0] {
		case "null":
			pathType = nil
		case "boolean":
			pathType = false
		case "number":
			fallthrough
		case "integer":
			pathType = float64(0)
		case "string":
			pathType = ""
		case "array":
			pathType = []interface{}{}
		case "object":
			// Only store paths for objects that have properties
			if len(schema.Properties) == 0 {
				pathType = map[string]interface{}{}
			}
		}
	}

	var def interface{} = schema.Default
	if v, ok := def.(json.Number); ok {
		def, _ = v.Float64()
	}
	if pathType != nil || schema.Default != nil {
		paths = append(paths, Path{
			Name:    strings.Join(parents, "."),
			Default: def,
			Type:    pathType,
		})
	}

	if schema.Ref != nil {
		path, err := listPaths(schema.Ref, parents, appendPointer(pointers, schema))
		if err != nil {
			return nil, err
		}
		paths = append(paths, path...)
	}

	if schema.Not != nil {
		path, err := listPaths(schema.Not, parents, appendPointer(pointers, schema))
		if err != nil {
			return nil, err
		}
		paths = append(paths, path...)
	}

	if schema.If != nil {
		path, err := listPaths(schema.If, parents, appendPointer(pointers, schema))
		if err != nil {
			return nil, err
		}
		paths = append(paths, path...)
	}

	if schema.Then != nil {
		path, err := listPaths(schema.Then, parents, appendPointer(pointers, schema))
		if err != nil {
			return nil, err
		}
		paths = append(paths, path...)
	}

	if schema.Else != nil {
		path, err := listPaths(schema.Then, parents, appendPointer(pointers, schema))
		if err != nil {
			return nil, err
		}
		paths = append(paths, path...)
	}

	for _, sub := range schema.AllOf {
		path, err := listPaths(sub, parents, appendPointer(pointers, schema))
		if err != nil {
			return nil, err
		}
		paths = append(paths, path...)
	}

	for _, sub := range schema.AnyOf {
		path, err := listPaths(sub, parents, appendPointer(pointers, schema))
		if err != nil {
			return nil, err
		}
		paths = append(paths, path...)
	}

	for _, sub := range schema.OneOf {
		path, err := listPaths(sub, parents, appendPointer(pointers, schema))
		if err != nil {
			return nil, err
		}
		paths = append(paths, path...)
	}

	for name, sub := range schema.Properties {
		path, err := listPaths(sub, append(parents, name), appendPointer(pointers, schema))
		if err != nil {
			return nil, err
		}
		paths = append(paths, path...)
	}

	return paths, nil
}
