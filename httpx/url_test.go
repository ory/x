package httpx_test

import (
	"crypto/tls"
	"github.com/ory/x/httpx"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ory/x/urlx"
)

func TestIncomingRequestURL(t *testing.T) {
	assert.EqualValues(t, httpx.IncomingRequestURL(&http.Request{
		URL: urlx.ParseOrPanic("/foo"), Host: "foobar", TLS: &tls.ConnectionState{},
	}).String(), "https://foobar/foo")
	assert.EqualValues(t, httpx.IncomingRequestURL(&http.Request{
		URL: urlx.ParseOrPanic("/foo"), Host: "foobar",
	}).String(), "http://foobar/foo")
	assert.EqualValues(t, httpx.IncomingRequestURL(&http.Request{
		URL: urlx.ParseOrPanic("/foo"), Host: "foobar", Header: http.Header{"X-Forwarded-Host": []string{"notfoobar"}, "X-Forwarded-Proto": {"https"}},
	}).String(), "https://notfoobar/foo")
}
