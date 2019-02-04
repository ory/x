package urlx

import (
	"github.com/bmizerany/assert"
	"net/url"
	"testing"
)

func TestCopyWithQuery(t *testing.T) {
	a, _ := url.Parse("https://google.com/foo?bar=baz")
	b := CopyWithQuery(a, url.Values{"foo": {"bar"}})
	assert.NotEqual(t, a.String(), b.String())
	assert.Equal(t, "bar", b.Query().Get("foo"))
}

func TestCopy(t *testing.T) {
	a, _ := url.Parse("https://google.com/foo?bar=baz")
	b := Copy(a)
	b.Path = "bar"
	assert.NotEqual(t, a.String(), b.String())
}
