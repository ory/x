package corsx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHelpMessage(t *testing.T) {
	assert.NotEmpty(t, HelpMessage())
}
