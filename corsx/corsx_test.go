package corsx

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHelpMessage(t *testing.T) {
	assert.NotEmpty(t, HelpMessage())
}
