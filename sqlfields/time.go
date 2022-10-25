package sqlfields

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
)

type Duration time.Duration

type Time time.Time

func NewNullTime(t time.Time) NullTime {
	return NullTime{Val: Time(t), Valid: true}
}

func (t *Time) Scan(value any) error {
	fmt.Printf("Scanning %#v\n", value)
	switch v := value.(type) {
	case time.Time:
		*t = Time(v)
	case string:
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return errors.WithStack(err)
		}
		*t = Time(parsed)
	default:
		return errors.Errorf("Time.Scan: cannot scan type %T into Time", value)
	}
	return nil
}

func (t Time) Value() (driver.Value, error) {
	fmt.Printf("Valuing %s\n", time.Time(t).Format(time.RFC3339))
	return time.Time(t).Format(time.RFC3339), nil
}

func (t Time) MarshalJSON() ([]byte, error) {
	return (time.Time)(t).UTC().MarshalJSON()
}

func (t *Time) UnmarshalJSON(data []byte) error {
	var st time.Time
	if err := json.Unmarshal(data, &st); err != nil {
		return err
	}
	*t = Time(st)
	return nil
}

func NewNullDuration(d time.Duration) NullDuration {
	return NullDuration{Val: Duration(d), Valid: true}
}

func (d *Duration) Scan(value any) error {
	switch v := value.(type) {
	case time.Duration:
		*d = Duration(v)
	case int64:
		*d = Duration(v)
	case string:
		parsed, err := time.ParseDuration(v)
		if err != nil {
			return errors.WithStack(err)
		}
		*d = Duration(parsed)
	default:
		return errors.Errorf("Duration.Scan: cannot scan type %T into Duration", value)
	}
	return nil
}

func (d Duration) Value() (driver.Value, error) {
	return int64(d), nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return errors.WithStack(err)
	}
	if len(s) == 0 {
		// set to zero value
		*d = 0
		return nil
	}

	p, err := time.ParseDuration(s)
	if err != nil {
		return errors.WithStack(err)
	}

	*d = Duration(p)
	return nil
}
