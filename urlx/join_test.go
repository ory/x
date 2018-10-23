package urlx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoin(t *testing.T) {
	assert.EqualValues(t, "http://foo/bar/baz/bar", MustJoin("http://foo", "bar/", "/baz", "bar"))
}
