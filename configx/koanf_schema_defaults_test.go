package configx

import (
	"bytes"
	"context"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ory/jsonschema/v3"
	"github.com/ory/x/snapshotx"
)

func TestKoanfSchemaDefaults(t *testing.T) {
	schemaPath := path.Join("stub", "domain-aliases", "config.schema.json")

	rawSchema, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	c := jsonschema.NewCompiler()
	require.NoError(t, c.AddResource(schemaPath, bytes.NewReader(rawSchema)))

	schema, err := c.Compile(context.Background(), schemaPath)
	require.NoError(t, err)

	k, err := newKoanf(ctx, schemaPath, nil)
	require.NoError(t, err)

	def, err := NewKoanfSchemaDefaults(rawSchema, schema)
	require.NoError(t, err)

	require.NoError(t, k.Load(def, nil))

	snapshotx.SnapshotTExcept(t, k.All(), nil)
}
