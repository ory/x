package contextx

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTreeContext(t *testing.T) {
	assert.True(t, IsRootContext(RootContext))
	assert.True(t, IsRootContext(context.WithValue(RootContext, "foo", "bar")))
	assert.False(t, IsRootContext(context.Background()))
}
