// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package keysetpagination

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
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
	require.Len(t, links, 2)
	assert.Contains(t, links[0], "page_token=default")
	assert.Contains(t, links[1], "page_token=next")

	t.Run("with isLast", func(t *testing.T) {
		p.isLast = true

		Header(r, u, p)

		links := r.HeaderMap["Link"]
		require.Len(t, links, 1)
		assert.Contains(t, links[0], "page_token=default")
	})

}
