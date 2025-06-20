// Copyright © 2025 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package keysetpagination

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageToken(t *testing.T) {
	t.Parallel()

	t.Run("json idempotency", func(t *testing.T) {
		token := NewPageToken(Column{Name: "id", Value: "token"}, Column{Name: "name", Order: OrderDescending, Value: "My Name"})
		raw, err := token.MarshalJSON()
		require.NoError(t, err)

		var decodedToken PageToken
		require.NoError(t, decodedToken.UnmarshalJSON(raw))

		assert.Equal(t, token, decodedToken)
	})

	t.Run("checks expiration", func(t *testing.T) {
		now := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		token := NewPageToken(Column{Name: "id", Value: "token"})
		token.testNow = func() time.Time { return now }

		raw, err := token.MarshalJSON()
		require.NoError(t, err)

		decodedToken := PageToken{
			testNow: func() time.Time { return now.Add(2 * time.Hour) },
		}
		assert.ErrorIs(t, decodedToken.UnmarshalJSON(raw), ErrPageTokenExpired)
	})
}
