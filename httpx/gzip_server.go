// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package httpx

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

type CompressionRequestReader struct {
	ErrHandler func(w http.ResponseWriter, r *http.Request, err error)
}

func defaultCompressionErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, err.Error(), http.StatusBadRequest)
}

func NewCompressionRequestReader(eh func(w http.ResponseWriter, r *http.Request, err error)) *CompressionRequestReader {
	if eh == nil {
		eh = defaultCompressionErrorHandler
	}

	return &CompressionRequestReader{
		ErrHandler: eh,
	}
}

func (c *CompressionRequestReader) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	for _, enc := range strings.Split(r.Header.Get("Content-Encoding"), ",") {
		switch enc = strings.TrimSpace(enc); enc {
		case "gzip":
			var b bytes.Buffer
			reader, err := gzip.NewReader(r.Body)
			if err != nil {
				c.ErrHandler(w, r, err)
				return
			}

			/* #nosec G110 - FIXME */
			if _, err := io.Copy(&b, reader); err != nil {
				c.ErrHandler(w, r, err)
				return
			}

			r.Body = ioutil.NopCloser(&b)
		case "identity":
			fallthrough
		case "":
			// nothing to do
		default:
			c.ErrHandler(w, r, fmt.Errorf("%s content encoding not supported", enc))
		}
	}

	next(w, r)
}
