package sqlxx

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/gofrs/uuid"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/ory/x/stringsx"
)

// StringSlicePipeDelimiter de/encodes the string slice to/from a SQL string.
type StringSlicePipeDelimiter []string

// Scan implements the Scanner interface.
func (n *StringSlicePipeDelimiter) Scan(value interface{}) error {
	var s sql.NullString
	if err := s.Scan(value); err != nil {
		return err
	}
	*n = scanStringSlice('|', s.String)
	return nil
}

// Value implements the driver Valuer interface.
func (n StringSlicePipeDelimiter) Value() (driver.Value, error) {
	return valueStringSlice('|', n), nil
}

func scanStringSlice(delimiter rune, value interface{}) []string {
	return stringsx.Splitx(fmt.Sprintf("%s", value), string(delimiter))
}

func valueStringSlice(delimiter rune, value []string) string {
	return strings.Join(value, string(delimiter))
}

// swagger:type string
type NullString string

// MarshalJSON returns m as the JSON encoding of m.
func (ns NullString) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(ns))
}

// UnmarshalJSON sets *m to a copy of data.
func (ns *NullString) UnmarshalJSON(data []byte) error {
	if ns == nil {
		return errors.New("json.RawMessage: UnmarshalJSON on nil pointer")
	}
	if len(data) == 0 {
		return nil
	}
	return errors.WithStack(json.Unmarshal(data, ns))
}

// Scan implements the Scanner interface.
func (ns *NullString) Scan(value interface{}) error {
	var v sql.NullString
	if err := (&v).Scan(value); err != nil {
		return err
	}
	*ns = NullString(v.String)
	return nil
}

// Value implements the driver Valuer interface.
func (ns NullString) Value() (driver.Value, error) {
	if len(ns) == 0 {
		return sql.NullString{}.Value()
	}
	return sql.NullString{Valid: true, String: string(ns)}.Value()
}

// String implements the Stringer interface.
func (ns NullString) String() string {
	return string(ns)
}

// NullTime implements sql.NullTime functionality.
type NullTime time.Time

// Scan implements the Scanner interface.
func (ns *NullTime) Scan(value interface{}) error {
	var v sql.NullTime
	if err := (&v).Scan(value); err != nil {
		return err
	}
	*ns = NullTime(v.Time)
	return nil
}

// MarshalJSON returns m as the JSON encoding of m.
func (ns NullTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(ns))
}

// UnmarshalJSON sets *m to a copy of data.
func (ns *NullTime) UnmarshalJSON(data []byte) error {
	var t time.Time
	if err := json.Unmarshal(data, &t); err != nil {
		return err
	}
	*ns = NullTime(t)
	return nil
}

// Value implements the driver Valuer interface.
func (ns NullTime) Value() (driver.Value, error) {
	return sql.NullTime{Valid: !time.Time(ns).IsZero(), Time: time.Time(ns)}.Value()
}

// MapStringInterface represents a map[string]interface that works well with JSON, SQL, and Swagger.
type MapStringInterface map[string]interface{}

// Scan implements the Scanner interface.
func (n *MapStringInterface) Scan(value interface{}) error {
	v := fmt.Sprintf("%s", value)
	if len(v) == 0 {
		return nil
	}
	return errors.WithStack(json.Unmarshal([]byte(v), n))
}

// Value implements the driver Valuer interface.
func (n MapStringInterface) Value() (driver.Value, error) {
	value, err := json.Marshal(n)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return string(value), nil
}

// JSONRawMessage represents a json.RawMessage that works well with JSON, SQL, and Swagger.
type JSONRawMessage json.RawMessage

// Scan implements the Scanner interface.
func (m *JSONRawMessage) Scan(value interface{}) error {
	*m = []byte(fmt.Sprintf("%s", value))
	return nil
}

// Value implements the driver Valuer interface.
func (m JSONRawMessage) Value() (driver.Value, error) {
	if len(m) == 0 {
		return "null", nil
	}
	return string(m), nil
}

// MarshalJSON returns m as the JSON encoding of m.
func (m JSONRawMessage) MarshalJSON() ([]byte, error) {
	if len(m) == 0 {
		return []byte("null"), nil
	}
	return m, nil
}

// UnmarshalJSON sets *m to a copy of data.
func (m *JSONRawMessage) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("json.RawMessage: UnmarshalJSON on nil pointer")
	}
	*m = append((*m)[0:0], data...)
	return nil
}

// NullJSONRawMessage represents a json.RawMessage that works well with JSON, SQL, and Swagger and is NULLable-
type NullJSONRawMessage json.RawMessage

// Scan implements the Scanner interface.
func (m *NullJSONRawMessage) Scan(value interface{}) error {
	if value == nil {
		value = "null"
	}
	*m = []byte(fmt.Sprintf("%s", value))
	return nil
}

// Value implements the driver Valuer interface.
func (m NullJSONRawMessage) Value() (driver.Value, error) {
	if len(m) == 0 {
		return nil, nil
	}
	return string(m), nil
}

// MarshalJSON returns m as the JSON encoding of m.
func (m NullJSONRawMessage) MarshalJSON() ([]byte, error) {
	if len(m) == 0 {
		return []byte("null"), nil
	}
	return m, nil
}

// UnmarshalJSON sets *m to a copy of data.
func (m *NullJSONRawMessage) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("json.RawMessage: UnmarshalJSON on nil pointer")
	}
	*m = append((*m)[0:0], data...)
	return nil
}

type NullUUID uuid.UUID

func (u *NullUUID) Scan(value interface{}) error {
	if value == nil {
		*u = NullUUID(uuid.Nil)
	}
	var uid uuid.UUID
	err := uid.Scan(value)
	*u = NullUUID(uid)
	return err
}

func (u *NullUUID) Value() (driver.Value, error) {
	if *u == NullUUID(uuid.Nil) {
		return nil, nil
	}
	return uuid.UUID(*u).Value()
}
