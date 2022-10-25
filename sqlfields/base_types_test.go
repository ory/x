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

func testJSONEncoding[T any](t *testing.T, value T, expectedJSON string, isNullable bool) {
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
			assert.NoError(t, json.Unmarshal([]byte("null"), &actual))
			if _, ok := any(value).(JSONRawMessage); ok {
				assert.Equal(t, JSONRawMessage("null"), actual)
			} else {
				assert.Equal(t, expected, actual)
			}

			raw, err := json.Marshal(expected)
			require.NoError(t, err)
			if !isNullable {
				assert.NotEqual(t, "null", string(raw))
			} else {
				assert.Equal(t, "null", string(raw))
			}
		})
	})
}

func TestJSONCompat(t *testing.T) {
	testJSONEncoding(t, String("foo"), `"foo"`, false)
	testJSONEncoding(t, Int(123), `123`, false)
	testJSONEncoding(t, Int32(456), `456`, false)
	testJSONEncoding(t, Int64(789), `789`, false)
	testJSONEncoding(t, Float64(1.23), `1.23`, false)
	testJSONEncoding(t, Bool(true), `true`, false)
	testJSONEncoding(t, Duration(10*time.Second), `"10s"`, false)
	testJSONEncoding(t, Time(time.Unix(123, 0).UTC()), `"1970-01-01T00:02:03Z"`, false)
	testJSONEncoding(t, JSONRawMessage(`{"foo":"bar"}`), `{"foo":"bar"}`, true)
	testJSONEncoding(t, StringSliceJSONFormat{"foo", "bar"}, `["foo","bar"]`, true)
	testJSONEncoding(t, StringSlicePipeDelimiter{"foo", "bar"}, `["foo","bar"]`, true)
}

func testSQLCompatibility[T any](t *testing.T, db *sqlx.DB, value T) {
	insertValue := func(t *testing.T, value T) int64 {
		res, err := db.Exec(`INSERT INTO "testing" ("value") VALUES (?)`, value)
		require.NoError(t, err)
		id, err := res.LastInsertId()
		require.NoError(t, err)
		return id
	}

	t.Run(fmt.Sprintf("type=%T", value), func(t *testing.T) {
		t.Run("case=insert and select", func(t *testing.T) {
			var actual T
			require.NoError(t, db.Get(&actual, `SELECT "value" FROM "testing" WHERE "id" = ?`, insertValue(t, value)))
			assert.EqualValues(t, value, actual)
		})
	})
}

func TestSQLCompat(t *testing.T) {
	db, err := sqlx.Connect("sqlite3", "file::memory:")
	require.NoError(t, err)
	defer db.Close()

	// You have to hate the inconsistencies of SQLite. But for this test, it's great to have column that takes any data type.
	_, err = db.Exec(`CREATE TABLE "testing" (
		"id" INTEGER PRIMARY KEY AUTOINCREMENT,
		"value" BLOB
)`)
	require.NoError(t, err)

	testSQLCompatibility(t, db, String("foo"))
	testSQLCompatibility(t, db, Int(123))
	testSQLCompatibility(t, db, Int32(456))
	testSQLCompatibility(t, db, Int64(789))
	testSQLCompatibility(t, db, Float64(1.23))
	testSQLCompatibility(t, db, Bool(true))
	testSQLCompatibility(t, db, Duration(10*time.Second))
	testSQLCompatibility(t, db, Time(time.Unix(12345, 0).UTC()))
	testSQLCompatibility(t, db, JSONRawMessage(`{"foo":"bar"}`))
	testSQLCompatibility(t, db, StringSliceJSONFormat{"foo", "bar"})
	testSQLCompatibility(t, db, StringSlicePipeDelimiter{"foo", "bar"})
}
