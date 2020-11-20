package configx

import (
	"bytes"
	"fmt"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"

	"github.com/ory/jsonschema/v3"
)

func newCompiler(schema []byte) (string, *jsonschema.Compiler, error) {
	id := gjson.GetBytes(schema, "$id").String()
	if id == "" {
		id = fmt.Sprintf("%s.json", uuid.New().String())
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(id, bytes.NewBuffer(schema)); err != nil {
		return "", nil, errors.WithStack(err)
	}
	compiler.ExtractAnnotations = true

	return id, compiler, nil
}
