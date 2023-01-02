// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package httpx

import (
	"mime"
	"net/http"
	"strings"

	"github.com/ory/x/stringslice"
)

// HasContentType determines whether the request `content-type` includes a
// server-acceptable mime-type
//
// Failure should yield an HTTP 415 (`http.StatusUnsupportedMediaType`)
func HasContentType(r *http.Request, mimetypes ...string) bool {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return stringslice.Has(mimetypes, "application/octet-stream")
	}

	for _, v := range strings.Split(contentType, ",") {
		t, _, err := mime.ParseMediaType(strings.TrimSpace(v))
		if err != nil {
			break
		}
		if stringslice.Has(mimetypes, t) {
			return true
		}
	}
	return false
}
