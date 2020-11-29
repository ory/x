package configx

import (
	"testing"

	"github.com/spf13/pflag"

	"github.com/bmizerany/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderMethods(t *testing.T) {
	// Fake some flags
	f := pflag.NewFlagSet("config", pflag.ContinueOnError)
	f.String("foo-bar-baz", "", "")
	f.StringP("b", "b", "", "")
	args := []string{"/var/folders/mt/m1dwr59n73zgsq7bk0q2lrmc0000gn/T/go-build533083141/b001/exe/asdf", "aaaa", "-b", "bbbb", "dddd", "eeee", "--foo-bar-baz", "fff"}
	require.NoError(t, f.Parse(args[1:]))
	RegisterFlags(f)

	p, err := New([]byte(`{}`), f)
	require.NoError(t, err)

	t.Run("check flags", func(t *testing.T) {
		assert.Equal(t, "fff", p.String("foo-bar-baz"))
		assert.Equal(t, "bbbb", p.String("b"))
	})

	t.Run("check fallbacks", func(t *testing.T) {
		t.Run("type=string", func(t *testing.T) {
			p.Set("some.string", "bar")
			assert.Equal(t, "bar", p.String("some.string"))
			assert.Equal(t, "bar", p.StringF("some.string", "baz"))
			assert.Equal(t, "baz", p.StringF("not.some.string", "baz"))
		})
	})
}
