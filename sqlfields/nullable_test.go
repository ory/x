package sqlfields

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testNullJSONEncoding[T any](t *testing.T, value T, expectedJSON string) {
	t.Run(fmt.Sprintf("type=%T", value), func(t *testing.T) {
		t.Run("case=marshal", func(t *testing.T) {
			actual, err := json.Marshal(value)
			require.NoError(t, err)
			assert.JSONEq(t, expectedJSON, string(actual))
		})
		t.Run("case=unmarshal", func(t *testing.T) {
			var other T
			require.NoError(t, json.Unmarshal([]byte(expectedJSON), &other))
			assert.EqualValues(t, value, other)
		})
		t.Run("case=null", func(t *testing.T) {
			var actual, expected T
			require.NoError(t, json.Unmarshal([]byte("null"), &actual))
			assert.EqualValues(t, expected, actual)

			raw, err := json.Marshal(expected)
			require.NoError(t, err)
			assert.JSONEq(t, "null", string(raw))
		})
	})
}

func TestNullableJSON(t *testing.T) {
	testNullJSONEncoding(t, NewNullString("foo"), `"foo"`)
	testNullJSONEncoding(t, NewNullInt(123), `123`)
	testNullJSONEncoding(t, NewNullInt32(456), `456`)
	testNullJSONEncoding(t, NewNullInt64(789), `789`)
	testNullJSONEncoding(t, NewNullFloat64(1.23), `1.23`)
	testNullJSONEncoding(t, NewNullBool(true), `true`)
	testNullJSONEncoding(t, NewNullDuration(10*time.Second), `"10s"`)
	testNullJSONEncoding(t, NewNullTime(time.Unix(123, 0).UTC()), `"1970-01-01T00:02:03Z"`)
	testNullJSONEncoding(t, NewNullJSONRawMessage([]byte(`{"foo":"bar"}`)), `{"foo":"bar"}`)
}

func testNullSQLCompatibility[T any](t *testing.T, db *sqlx.DB, value T) {
	insertValue := func(t *testing.T, value T) int64 {
		res, err := db.Exec(`INSERT INTO "testing" ("value") VALUES (?)`, value)
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		return id
	}

	t.Run(fmt.Sprintf("type=%T", value), func(t *testing.T) {
		t.Run("case=insert and select non-null values", func(t *testing.T) {
			var actual T
			require.NoError(t, db.Get(&actual, `SELECT "value" FROM "testing" WHERE "id" = ?`, insertValue(t, value)))
			assert.EqualValues(t, value, actual)
		})

		t.Run("case=insert and select null values", func(t *testing.T) {
			var actual, null T
			require.NoError(t, db.Get(&actual, `SELECT "value" FROM "testing" WHERE "id" = ?`, insertValue(t, null)))
			assert.Equal(t, null, actual)
		})
	})
}

func TestNullableSQL(t *testing.T) {
	db, err := sqlx.Connect("sqlite3", "file::memory:?cache=shared")
	require.NoError(t, err)
	defer db.Close()

	// You have to hate the inconsistencies of SQLite. But for this test, it's great to have column that takes any data type.
	_, err = db.Exec(`CREATE TABLE "testing" (
		"id" INTEGER PRIMARY KEY AUTOINCREMENT,
		"value" BLOB
)`)
	require.NoError(t, err)

	testNullSQLCompatibility(t, db, NewNullString("foo"))
	testNullSQLCompatibility(t, db, NewNullInt(123))
	testNullSQLCompatibility(t, db, NewNullInt32(456))
	testNullSQLCompatibility(t, db, NewNullInt64(789))
	testNullSQLCompatibility(t, db, NewNullFloat64(1.23))
	testNullSQLCompatibility(t, db, NewNullBool(true))
	testNullSQLCompatibility(t, db, NewNullDuration(10*time.Second))
	testNullSQLCompatibility(t, db, NewNullJSONRawMessage([]byte(`{"foo":"bar"}`)))
}
