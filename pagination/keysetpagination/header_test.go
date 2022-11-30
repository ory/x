// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package keysetpagination

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/instana/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeader(t *testing.T) {
	p := &Paginator{
		defaultToken: StringPageToken("default"),
		token:        StringPageToken("next"),
		size:         2,
	}

	u, err := url.Parse("http://ory.sh/")
	require.NoError(t, err)

	r := httptest.NewRecorder()

	Header(r, u, p)

	links := r.HeaderMap["Link"]
	assert.Len(t, links, 2)
	assert.Contains(t, links[0], "page_token=default")
	assert.Contains(t, links[1], "page_token=next")
}

func TestHeader_WithIsLast(t *testing.T) {
	p := &Paginator{
		defaultToken: StringPageToken("default"),
		token:        StringPageToken("next"),
		size:         2,
		isLast:       true,
	}

	u, err := url.Parse("http://ory.sh/")
	require.NoError(t, err)

	r := httptest.NewRecorder()

	Header(r, u, p)

	links := r.HeaderMap["Link"]
	assert.Len(t, links, 1)
	assert.Contains(t, links[0], "page_token=default")

	t.Logf("%v", url.QueryEscape("pk=token/created_at=123"))
}
