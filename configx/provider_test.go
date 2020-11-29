package configx

import (
	"testing"

	"github.com/ory/x/urlx"

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
			assert.Equal(t, "bar", p.StringF("some.string", "baz"))
			assert.Equal(t, "baz", p.StringF("not.some.string", "baz"))
		})
		t.Run("type=float", func(t *testing.T) {
			p.Set("some.float", 123.123)
			assert.Equal(t, 123.123, p.Float64F("some.float", 321.321))
			assert.Equal(t, 321.321, p.Float64F("not.some.float", 321.321))
		})
		t.Run("type=int", func(t *testing.T) {
			p.Set("some.int", 123)
			assert.Equal(t, 123, p.IntF("some.int", 123))
			assert.Equal(t, 321, p.IntF("not.some.int", 321))
		})

		github := urlx.ParseOrPanic("https://github.com/ory")
		ory := urlx.ParseOrPanic("https://www.ory.sh/")

		t.Run("type=url", func(t *testing.T) {
			p.Set("some.url", "https://github.com/ory")
			assert.Equal(t, github, p.URIF("some.url", ory))
			assert.Equal(t, ory, p.URIF("not.some.url", ory))
		})

		t.Run("type=request_uri", func(t *testing.T) {
			p.Set("some.request_uri", "https://github.com/ory")
			assert.Equal(t, github, p.RequestURIF("some.request_uri", ory))
			assert.Equal(t, ory, p.RequestURIF("not.some.request_uri", ory))

			p.Set("invalid.request_uri", "foo")
			assert.Equal(t, ory, p.RequestURIF("invalid.request_uri", ory))
		})
	})
}
