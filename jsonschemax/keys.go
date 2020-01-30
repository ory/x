package jsonschemax

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/santhosh-tekuri/jsonschema/v2"

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

	// Format is the format of the path if defined
	Format string

	// Pattern is the pattern of the path if defined
	Pattern *regexp.Regexp

	// Enum are the allowed enum values
	Enum []interface{}

	// first element in slice is constant value. note: slice is used to capture nil constant.
	Constant []interface{}

	// Required is whether the value is required
	// TODO
	Required bool

	// -1 if not specified
	MinLength int
	MaxLength int

	Minimum *big.Float
	Maximum *big.Float

	MultipleOf *big.Float
}

// ListPathsBytes works like ListPathsWithRecursion but prepares the JSON Schema itself.
func ListPathsBytes(raw json.RawMessage, maxRecursion int16) ([]Path, error) {
	compiler := jsonschema.NewCompiler()
	compiler.ExtractAnnotations = true
	id := fmt.Sprintf("%x.json", sha256.Sum256(raw))
	if err := compiler.AddResource(id, bytes.NewReader(raw)); err != nil {
		return nil, err
	}
	compiler.ExtractAnnotations = true
	return runPaths(id, compiler, maxRecursion)
}

// ListPathsWithRecursion will follow circular references until maxRecursion is reached, without
// returning an error.
func ListPathsWithRecursion(ref string, compiler *jsonschema.Compiler, maxRecursion uint8) ([]Path, error) {
	return runPaths(ref, compiler, int16(maxRecursion))
}

// ListPaths lists all paths of a JSON Schema. Will return an error
// if circular references are found.
func ListPaths(ref string, compiler *jsonschema.Compiler) ([]Path, error) {
	return runPaths(ref, compiler, -1)
}

func runPaths(ref string, compiler *jsonschema.Compiler, maxRecursion int16) ([]Path, error) {
	if compiler == nil {
		compiler = jsonschema.NewCompiler()
	}

	compiler.ExtractAnnotations = true
	pointers := map[string]bool{}

	schema, err := compiler.Compile(ref)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	paths, err := listPaths(schema, nil, pointers, 0, maxRecursion)
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

func listPaths(schema *jsonschema.Schema, parents []string, pointers map[string]bool, currentRecursion int16, maxRecursion int16) (byName, error) {
	var pathType interface{}
	var paths []Path
	_, isCircular := pointers[fmt.Sprintf("%p", schema)]

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
	} else if len(schema.Types) == 1 {
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
			if schema.Items != nil {
				var types []string
				switch t := schema.Items.(type) {
				case []*jsonschema.Schema:
					for _, tt := range t {
						types = append(types, tt.Types...)
					}
				case *jsonschema.Schema:
					types = append(types, t.Types...)
				}
				types = stringslice.Unique(types)
				if len(types) == 1 {
					switch types[0] {
					case "boolean":
						pathType = []bool{}
					case "number":
						fallthrough
					case "integer":
						pathType = []float64{}
					case "string":
						pathType = []string{}
					}
				}
			}
		case "object":
			// Only store paths for objects that have properties
			if len(schema.Properties) == 0 {
				pathType = map[string]interface{}{}
			}
		}
	} else if len(schema.Types) > 2 {
		pathType = nil
	}

	var def interface{} = schema.Default
	if v, ok := def.(json.Number); ok {
		def, _ = v.Float64()
	}
	if (pathType != nil || schema.Default != nil) && len(parents) > 0 {
		paths = append(paths, Path{
			Name:       strings.Join(parents, "."),
			Default:    def,
			Type:       pathType,
			Format:     schema.Format,
			Pattern:    schema.Pattern,
			Enum:       schema.Enum,
			Constant:   schema.Constant,
			MinLength:  schema.MinLength,
			MaxLength:  schema.MaxLength,
			Minimum:    schema.Minimum,
			Maximum:    schema.Maximum,
			MultipleOf: schema.MultipleOf,
		})
	}

	if isCircular {
		if maxRecursion == -1 {
			return nil, errors.Errorf("detected circular dependency in schema path: %s", strings.Join(parents, "."))
		} else if currentRecursion > maxRecursion {
			return paths, nil
		}
		currentRecursion++
	}

	if schema.Ref != nil {
		path, err := listPaths(schema.Ref, parents, appendPointer(pointers, schema), currentRecursion, maxRecursion)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path...)
	}

	if schema.Not != nil {
		path, err := listPaths(schema.Not, parents, appendPointer(pointers, schema), currentRecursion, maxRecursion)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path...)
	}

	if schema.If != nil {
		path, err := listPaths(schema.If, parents, appendPointer(pointers, schema), currentRecursion, maxRecursion)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path...)
	}

	if schema.Then != nil {
		path, err := listPaths(schema.Then, parents, appendPointer(pointers, schema), currentRecursion, maxRecursion)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path...)
	}

	if schema.Else != nil {
		path, err := listPaths(schema.Else, parents, appendPointer(pointers, schema), currentRecursion, maxRecursion)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path...)
	}

	for _, sub := range schema.AllOf {
		path, err := listPaths(sub, parents, appendPointer(pointers, schema), currentRecursion, maxRecursion)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path...)
	}

	for _, sub := range schema.AnyOf {
		path, err := listPaths(sub, parents, appendPointer(pointers, schema), currentRecursion, maxRecursion)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path...)
	}

	for _, sub := range schema.OneOf {
		path, err := listPaths(sub, parents, appendPointer(pointers, schema), currentRecursion, maxRecursion)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path...)
	}

	for name, sub := range schema.Properties {
		path, err := listPaths(sub, append(parents, name), appendPointer(pointers, schema), currentRecursion, maxRecursion)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path...)
	}

	return paths, nil
}
