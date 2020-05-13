package sqlxx

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

type RawJSON json.RawMessage

func (t *RawJSON) Scan(value interface{}) error {
	return JSONScan(t, value)
}

func (t *RawJSON) Value() (driver.Value, error) {
	return JSONValue(t)
}

func JSONScan(dst interface{}, value interface{}) error {
	var b bytes.Buffer
	switch v := value.(type) {
	case []byte:
		b.Write(v)
	case string:
		b.WriteString(v)
	default:
		return errors.Errorf("unable to decode value of type: %T %v", value, value)
	}

	if err := json.NewDecoder(&b).Decode(&dst); err != nil {
		return fmt.Errorf("unable to decode payload to: %s", err)
	}

	return nil
}

func JSONValue(src interface{}) (driver.Value, error) {
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(&src); err != nil {
		return nil, err
	}
	return b.String(), nil
}
