package sqlfields

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/pkg/errors"
)

func NewNullJSONRawMessage(data []byte) NullJSONRawMessage {
	if data == nil {
		return NullJSONRawMessage{}
	}
	return NullJSONRawMessage{Val: data, Valid: true}
}

type JSONRawMessage json.RawMessage

func (j *JSONRawMessage) Scan(value any) error {
	switch v := value.(type) {
	case []byte:
		*j = v
	case string:
		*j = JSONRawMessage(v)
	default:
		return errors.Errorf("JSONRawMessage.Scan: cannot scan type %T into JSONRawMessage", value)
	}
	return nil
}

func (j *JSONRawMessage) Value() (driver.Value, error) {
	return []byte(*j), nil
}

func (j JSONRawMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(json.RawMessage(j))
}

func (j *JSONRawMessage) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*json.RawMessage)(j))
}
