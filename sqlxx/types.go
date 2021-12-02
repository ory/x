package sqlxx

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	"github.com/pkg/errors"
)

// StringSliceJSONFormat represents []string{} which is encoded to/from JSON for SQL storage.
type StringSliceJSONFormat []string

// Scan implements the Scanner interface.
func (m *StringSliceJSONFormat) Scan(value interface{}) error {
	val := fmt.Sprintf("%s", value)
	if len(val) == 0 {
		val = "[]"
	}

	if parsed := gjson.Parse(val); !parsed.IsArray() {
		return errors.Errorf("expected JSON value to be an array but got type: %s", parsed.Type.String())
	}

	return errors.WithStack(json.Unmarshal([]byte(val), &m))
}

// Value implements the driver Valuer interface.
func (m StringSliceJSONFormat) Value() (driver.Value, error) {
	if len(m) == 0 {
		return "[]", nil
	}

	encoded, err := json.Marshal(&m)
	return string(encoded), errors.WithStack(err)
}

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
	escaped := false
	s := fmt.Sprintf("%s", value)
	splitted := strings.FieldsFunc(s, func(r rune) bool {
		if r == '\\' {
			escaped = !escaped
		} else if escaped && r != delimiter {
			escaped = false
		}
		return !escaped && r == delimiter
	})
	for k, v := range splitted {
		splitted[k] = strings.ReplaceAll(v, "\\"+string(delimiter), string(delimiter))
	}
	return splitted
}

func valueStringSlice(delimiter rune, value []string) string {
	replace := make([]string, len(value))
	for k, v := range value {
		replace[k] = strings.ReplaceAll(v, string(delimiter), "\\"+string(delimiter))
	}
	return strings.Join(replace, string(delimiter))
}

// swagger:type bool
type NullBool sql.NullBool

// MarshalJSON returns m as the JSON encoding of m.
func (ns NullBool) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(ns.Bool)
}

// UnmarshalJSON sets *m to a copy of data.
func (ns *NullBool) UnmarshalJSON(data []byte) error {
	if ns == nil {
		return errors.New("json.RawMessage: UnmarshalJSON on nil pointer")
	}
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	ns.Valid = true
	return errors.WithStack(json.Unmarshal(data, &ns.Bool))
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
	return errors.WithStack(json.Unmarshal(data, (*string)(ns)))
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
//
// swagger:model nullTime
// required: false
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
	var t *time.Time
	if !time.Time(ns).IsZero() {
		tt := time.Time(ns)
		t = &tt
	}
	return json.Marshal(t)
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

// JSONArrayRawMessage represents a json.RawMessage which only accepts arrays that works well with JSON, SQL, and Swagger.
type JSONArrayRawMessage json.RawMessage

// Scan implements the Scanner interface.
func (m *JSONArrayRawMessage) Scan(value interface{}) error {
	val := fmt.Sprintf("%s", value)
	if len(val) == 0 {
		val = "[]"
	}

	if parsed := gjson.Parse(val); !parsed.IsArray() {
		return errors.Errorf("expected JSON value to be an array but got type: %s", parsed.Type.String())
	}

	*m = []byte(val)
	return nil
}

// Value implements the driver Valuer interface.
func (m JSONArrayRawMessage) Value() (driver.Value, error) {
	if len(m) == 0 {
		return "[]", nil
	}

	if parsed := gjson.ParseBytes(m); !parsed.IsArray() {
		return nil, errors.Errorf("expected JSON value to be an array but got type: %s", parsed.Type.String())
	}

	return string(m), nil
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
//
// swagger:model nullJsonRawMessage
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

// JSONScan is a generic helper for storing a value as a JSON blob in SQL.
func JSONScan(dst interface{}, value interface{}) error {
	if value == nil {
		value = "null"
	}
	if err := json.Unmarshal([]byte(fmt.Sprintf("%s", value)), &dst); err != nil {
		return fmt.Errorf("unable to decode payload to: %s", err)
	}
	return nil
}

// JSONValue is a generic helper for retrieving a SQL JSON-encoded value.
func JSONValue(src interface{}) (driver.Value, error) {
	if src == nil {
		return nil, nil
	}
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(&src); err != nil {
		return nil, err
	}
	return b.String(), nil
}
