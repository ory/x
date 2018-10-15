package cmdx

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEnvVarExamplesHelpMessage(t *testing.T) {
	assert.NotEmpty(t, EnvVarExamplesHelpMessage(""))
}
