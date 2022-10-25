package sqlfields

import (
	"database/sql/driver"
	"math"

	"github.com/pkg/errors"
)

func NewNullString(s string) NullString {
	return NullString{Val: String(s), Valid: true}
}

type String string

func (s *String) Scan(value any) error {
	switch v := value.(type) {
	case string:
		*s = String(v)
	case []byte:
		*s = String(v)
	default:
		return errors.Errorf("String.Scan: cannot scan type %T into String", value)
	}
	return nil
}

func (s *String) Value() (driver.Value, error) {
	return string(*s), nil
}

func NewNullInt64(i int64) NullInt64 {
	return NullInt64{Val: Int64(i), Valid: true}
}

type Int64 int64

func (i *Int64) Scan(value any) error {
	switch v := value.(type) {
	case int64:
		*i = Int64(v)
	case float64:
		*i = Int64(v)
	default:
		return errors.Errorf("Int64.Scan: cannot scan type %T into Int64", value)
	}
	return nil
}

func (i *Int64) Value() (driver.Value, error) {
	return int64(*i), nil
}

func NewNullInt32(i int32) NullInt32 {
	return NullInt32{Val: Int32(i), Valid: true}
}

type Int32 int32

func (i *Int32) Scan(value any) error {
	var i64 Int64
	if err := i64.Scan(value); err != nil {
		return err
	}
	if i64 > math.MaxInt32 {
		return errors.Errorf("Int32.Scan: value %x does not fit into int32", i64)
	}
	*i = Int32(i64)
	return nil
}

func (i *Int32) Value() (driver.Value, error) {
	return int64(*i), nil
}

func NewNullInt(i int) NullInt {
	return NullInt{Val: Int(i), Valid: true}
}

type Int int

func (i *Int) Scan(value any) error {
	var i64 Int64
	if err := i64.Scan(value); err != nil {
		return err
	}
	if i64 > math.MaxInt {
		return errors.Errorf("Int.Scan: value %x does not fit into int", value)
	}
	*i = Int(i64)
	return nil
}

func (i *Int) Value() (driver.Value, error) {
	return int64(*i), nil
}

func NewNullFloat64(f float64) NullFloat64 {
	return NullFloat64{Val: Float64(f), Valid: true}
}

type Float64 float64

func (f *Float64) Scan(value any) error {
	switch v := value.(type) {
	case float64:
		*f = Float64(v)
	case int64:
		*f = Float64(v)
	default:
		return errors.Errorf("Float64.Scan: cannot scan type %T into Float64", value)
	}
	return nil
}

func (f *Float64) Value() (driver.Value, error) {
	return float64(*f), nil
}

func NewNullBool(b bool) NullBool {
	return NullBool{Val: Bool(b), Valid: true}
}

type Bool bool

func (b *Bool) Scan(value any) error {
	switch v := value.(type) {
	case bool:
		*b = Bool(v)
	case int64:
		*b = v != 0
	default:
		return errors.Errorf("Bool.Scan: cannot scan type %T into Bool", value)
	}
	return nil
}

func (b *Bool) Value() (driver.Value, error) {
	return bool(*b), nil
}
