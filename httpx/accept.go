package httpx

import (
	"github.com/ory/x/stringslice"
	"mime"
	"net/http"
)

// Accepts determines whether the request `accept` includes a
// acceptable mime-type.
//
// Failure should yield an HTTP 415 (`http.StatusUnsupportedMediaType`)
func Accepts(r *http.Request, mimetypes ...string) bool {
	contentType := r.Header.Get("Accept")
	if contentType == "" {
		return stringslice.Has(mimetypes, "application/octet-stream")
	}

	t, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}

	if stringslice.Has(mimetypes, t) {
		return true
	}

	return false
}
