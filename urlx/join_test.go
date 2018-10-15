package urlx

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestJoin(t *testing.T) {
	assert.EqualValues(t, "http://foo/bar/baz/bar", MustJoin("http://foo", "bar/", "/baz", "bar"))
}
