package cloudx

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ory/x/assertx"
)

func TestReadConfigFiles(t *testing.T) {
	configs, err := ReadConfigFiles([]string{
		"fixtures/config/a.yaml",
		"fixtures/config/b.yml",
		"fixtures/config/c.json",
	})
	require.NoError(t, err)
	assertx.EqualAsJSON(t, json.RawMessage(`[{"a":true},{"b":true},{"c":true}]`), configs)
}
