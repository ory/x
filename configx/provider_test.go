package configx

import (
	"testing"

	"github.com/bmizerany/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderMethods(t *testing.T) {
	p, err := New([]byte(`{}`))
	require.NoError(t, err)

	t.Run("type=string", func(t *testing.T) {
		p.Set("string", "bar")
		assert.Equal(t, "bar", p.String("some.string"))
		assert.Equal(t, "bar", p.StringF("some.string", "baz"))
		assert.Equal(t, "baz", p.StringF("not.some.string", "baz"))
	})
}
