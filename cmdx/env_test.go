package cmdx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvVarExamplesHelpMessage(t *testing.T) {
	assert.NotEmpty(t, EnvVarExamplesHelpMessage(""))
}
