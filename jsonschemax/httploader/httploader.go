// Copyright 2017 Santhosh Kumar Tekuri. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package httploader implements loader.Loader for http/https url.
//
// The package is typically only imported for the side effect of
// registering its Loaders.
//
// To use httploader, link this package into your program:
//	import _ "github.com/santhosh-tekuri/jsonschema/httploader"
//
package httploader

import (
	"fmt"
	"io"
	"net/http"

	"github.com/santhosh-tekuri/jsonschema/v2"

	"github.com/ory/x/httpx"
)

// Load implements jsonschemav2.Loader
func Load(url string) (io.ReadCloser, error) {
	resp, err := httpx.NewResilientClientLatencyToleranceMedium(nil).Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("%s returned status code %d", url, resp.StatusCode)
	}
	return resp.Body, nil
}

func init() {
	jsonschema.Loaders["http"] = Load
	jsonschema.Loaders["https"] = Load
}
