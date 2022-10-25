package sqlfields

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"

	"github.com/pkg/errors"
)

type Nullable[T any, pointer interface {
	*T
	sql.Scanner
	driver.Valuer
}] struct {
	Val   T
	Valid bool
}

// swagger:type string
// swagger:x-nullable true
type NullString = Nullable[String, *String]

// swagger:type integer
// swagger:x-nullable true
type NullInt = Nullable[Int, *Int]

// swagger:type integer
// swagger:x-nullable true
type NullInt32 = Nullable[Int32, *Int32]

// swagger:type integer
// swagger:x-nullable true
type NullInt64 = Nullable[Int64, *Int64]

// swagger:type number
// swagger:x-nullable true
type NullFloat64 = Nullable[Float64, *Float64]

// swagger:type boolean
// swagger:x-nullable true
type NullBool = Nullable[Bool, *Bool]

// swagger:type object
// swagger:x-nullable true
type NullJSONRawMessage = Nullable[JSONRawMessage, *JSONRawMessage]

// swagger:type string
// swagger:x-nullable true
type NullDuration = Nullable[Duration, *Duration]

// swagger:type string
// swagger:x-nullable true
type NullTime = Nullable[Time, *Time]

func (n Nullable[T, pointer]) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Val)
}

func (n *Nullable[T, pointer]) UnmarshalJSON(data []byte) error {
	if n == nil {
		return errors.New("Nullable: UnmarshalJSON on nil pointer")
	}
	if len(data) == 0 || string(data) == "null" {
		var zero T
		n.Val, n.Valid = zero, false
		return nil
	}
	err := json.Unmarshal(data, &n.Val)
	if err != nil {
		return errors.WithStack(err)
	}
	n.Valid = true
	return nil
}

func (n *Nullable[T, pointer]) Scan(value any) error {
	if value == nil {
		var zero T
		n.Val, n.Valid = zero, false
		return nil
	}
	pValue := any(&n.Val).(pointer)
	if err := pValue.Scan(value); err != nil {
		return errors.WithStack(err)
	}
	n.Valid = true
	return nil
}

func (n Nullable[T, pointer]) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return any(&(n.Val)).(pointer).Value()
}
