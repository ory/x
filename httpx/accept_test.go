package httpx

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccepts(t *testing.T) {
	assert.True(t, Accepts(&http.Request{Header: map[string][]string{}}, "application/octet-stream"))
	assert.False(t, Accepts(&http.Request{Header: map[string][]string{}}, "not-application/octet-stream"))
	assert.True(t, Accepts(&http.Request{Header: map[string][]string{"Accept": {"application/octet-stream"}}}, "application/octet-stream"))
	assert.True(t, Accepts(&http.Request{Header: map[string][]string{"Accept": {"application/octet-stream, not-application/application"}}}, "not-application/application"))
	assert.True(t, Accepts(&http.Request{Header: map[string][]string{"Accept": {"application/octet-stream,not-application/application"}}}, "not-application/application"))
	assert.False(t, Accepts(&http.Request{Header: map[string][]string{"Accept": {"application/octet-stream, application/not-application"}}}, "not-application/not-octet-stream"))
	assert.False(t, Accepts(&http.Request{Header: map[string][]string{"Accept": {"a"}}}, "not-application/not-octet-stream"))
}
