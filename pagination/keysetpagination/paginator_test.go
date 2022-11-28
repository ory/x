// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package keysetpagination

import (
	"net/url"
	"strconv"
	"testing"

	"github.com/gobuffalo/pop/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testItem struct {
	ID        string `db:"pk"`
	CreatedAt string `db:"created_at"`
}

func (t testItem) PageToken() string {
	return t.ID
}

func TestPaginator(t *testing.T) {
	t.Run("paginates correctly", func(t *testing.T) {
		c, err := pop.NewConnection(&pop.ConnectionDetails{
			URL: "postgres://foo.bar",
		})
		require.NoError(t, err)
		q := pop.Q(c)
		paginator := GetPaginator(WithSize(10), WithToken("token"))
		q = q.Scope(Paginate[testItem](paginator))

		sql, args := q.ToSQL(&pop.Model{Value: new(testItem)})
		assert.Equal(t, "SELECT test_items.created_at, test_items.pk FROM test_items AS test_items WHERE \"pk\" > $1 ORDER BY \"pk\" ASC LIMIT 11", sql)
		assert.Equal(t, []interface{}{"token"}, args)
	})

	t.Run("paginates correctly mysql", func(t *testing.T) {
		c, err := pop.NewConnection(&pop.ConnectionDetails{
			URL: "mysql://user:pass@(host:1337)/database",
		})
		require.NoError(t, err)
		q := pop.Q(c)
		paginator := GetPaginator(WithSize(10), WithToken("token"))
		q = q.Scope(Paginate[testItem](paginator))

		sql, args := q.ToSQL(&pop.Model{Value: new(testItem)})
		assert.Equal(t, "SELECT test_items.created_at, test_items.pk FROM test_items AS test_items WHERE `pk` > ? ORDER BY `pk` ASC LIMIT 11", sql)
		assert.Equal(t, []interface{}{"token"}, args)
	})

	t.Run("returns correct result", func(t *testing.T) {
		items := []testItem{
			{ID: "1"},
			{ID: "2"},
			{ID: "3"},
			{ID: "4"},
			{ID: "5"},
			{ID: "6"},
			{ID: "7"},
			{ID: "8"},
			{ID: "9"},
			{ID: "10"},
			{ID: "11"},
		}
		paginator := GetPaginator(WithDefaultSize(10), WithToken("token"))
		items, nextPage := Result(items, paginator)
		assert.Len(t, items, 10)
		assert.Equal(t, "10", nextPage.Token())
		assert.Equal(t, 10, nextPage.Size())
	})

	t.Run("returns correct size and token", func(t *testing.T) {
		for _, tc := range []struct {
			name          string
			opts          []Option
			expectedSize  int
			expectedToken string
		}{
			{
				name:          "default",
				opts:          nil,
				expectedSize:  100,
				expectedToken: "",
			},
			{
				name:          "with size and token",
				opts:          []Option{WithSize(10), WithToken("token")},
				expectedSize:  10,
				expectedToken: "token",
			},
			{
				name:          "with custom defaults",
				opts:          []Option{WithDefaultSize(10), WithDefaultToken("token")},
				expectedSize:  10,
				expectedToken: "token",
			},
			{
				name:          "with custom defaults and size and token",
				opts:          []Option{WithDefaultSize(10), WithDefaultToken("token"), WithSize(20), WithToken("token2")},
				expectedSize:  20,
				expectedToken: "token2",
			},
			{
				name:         "with size and custom default and max size",
				opts:         []Option{WithSize(10), WithDefaultSize(20), WithMaxSize(5)},
				expectedSize: 5,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				paginator := GetPaginator(tc.opts...)
				assert.Equal(t, tc.expectedSize, paginator.Size())
				assert.Equal(t, tc.expectedToken, paginator.Token())
			})
		}
	})
}

func TestParse(t *testing.T) {
	for _, tc := range []struct {
		name          string
		q             url.Values
		expectedSize  int
		expectedToken string
	}{
		{
			name:          "with page token",
			q:             url.Values{"page_token": {"token3"}},
			expectedSize:  100,
			expectedToken: "token3",
		},
		{
			name:         "with page size",
			q:            url.Values{"page_size": {"123"}},
			expectedSize: 123,
		},
		{
			name:          "with page size and page token",
			q:             url.Values{"page_size": {"123"}, "page_token": {"token5"}},
			expectedSize:  123,
			expectedToken: "token5",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts, err := Parse(tc.q)
			require.NoError(t, err)
			paginator := GetPaginator(opts...)
			assert.Equal(t, tc.expectedSize, paginator.Size())
			assert.Equal(t, tc.expectedToken, paginator.Token())
		})
	}

	t.Run("invalid page size leads to err", func(t *testing.T) {
		_, err := Parse(url.Values{"page_size": {"invalid-int"}})
		require.ErrorIs(t, err, strconv.ErrSyntax)
	})
}

func TestPaginateWithAdditionalColumn(t *testing.T) {
	c, err := pop.NewConnection(&pop.ConnectionDetails{
		URL: "postgres://foo.bar",
	})
	require.NoError(t, err)

	for _, tc := range []struct {
		d    string
		opts []Option
		e    string
		args []interface{}
	}{
		{
			d:    "with sort by created_at DESC",
			opts: []Option{WithToken("pk=token_value/created_at=timestamp"), WithColumn("created_at", "DESC")},
			e:    "created_at\" < $1 OR (\"created_at\" = $2 AND \"pk\" > $3) ORDER BY \"created_at\" DESC, \"pk\" ASC",
			args: []interface{}{"timestamp", "timestamp", "token_value"},
		},
		{
			d:    "with sort by created_at ASC",
			opts: []Option{WithToken("pk=token_value/created_at=timestamp"), WithColumn("created_at", "ASC")},
			e:    "created_at\" > $1 OR (\"created_at\" = $2 AND \"pk\" > $3) ORDER BY \"created_at\" ASC, \"pk\" ASC",
			args: []interface{}{"timestamp", "timestamp", "token_value"},
		},
		{
			d:    "with malformed token",
			opts: []Option{WithToken("some/random/token"), WithColumn("created_at", "ASC")},
			e:    "WHERE \"pk\" > $1 ORDER BY \"pk\"",
			args: []interface{}{"some/random/token"},
		},
		{
			d:    "with unknown column",
			opts: []Option{WithToken("pk=token_value/created_at=timestamp"), WithColumn("unknown_column", "ASC")},
			e:    "WHERE \"pk\" > $1 ORDER BY \"pk\"",
			args: []interface{}{"pk=token_value/created_at=timestamp"},
		},
		{
			d:    "with unknown order",
			opts: []Option{WithToken("pk=token_value/created_at=timestamp"), WithColumn("created_at", Order("unknown order"))},
			e:    "WHERE \"pk\" > $1 ORDER BY \"pk\"",
			args: []interface{}{"pk=token_value/created_at=timestamp"},
		},
	} {
		t.Run("case="+tc.d, func(t *testing.T) {
			opts := append(tc.opts, WithSize(10))
			paginator := GetPaginator(opts...)
			sql, args := pop.Q(c).
				Scope(Paginate[testItem](paginator)).
				ToSQL(&pop.Model{Value: new(testItem)})
			assert.Contains(t, sql, tc.e)
			assert.Contains(t, sql, "LIMIT 11")
			assert.Equal(t, tc.args, args)
		})
	}
}
