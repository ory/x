// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package configx

import (
	"bytes"
	"fmt"

	"github.com/ory/x/logrusx"
	"github.com/ory/x/otelx"

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

	// DO NOT REMOVE THIS
	compiler.ExtractAnnotations = true

	if err := otelx.AddConfigSchema(compiler); err != nil {
		return "", nil, err
	}
	if err := logrusx.AddConfigSchema(compiler); err != nil {
		return "", nil, err
	}

	return id, compiler, nil
}
