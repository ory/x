package sqlfields

import (
	"database/sql/driver"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
)

type StringSliceJSONFormat []string

type StringSlicePipeDelimiter []string

func (s *StringSlicePipeDelimiter) Scan(value any) error {
	switch v := value.(type) {
	case string:
		*s = scanStringSlice('|', v)
	case []byte:
		*s = scanStringSlice('|', string(v))
	default:
		return errors.Errorf("StringSlicePipeDelimiter.Scan: cannot scan type %T into StringSlicePipeDelimiter", value)
	}
	return nil
}

func (s StringSlicePipeDelimiter) Value() (driver.Value, error) {
	return valueStringSlice('|', s), nil
}

func scanStringSlice(delimiter rune, value string) []string {
	escaped := false
	splitted := strings.FieldsFunc(value, func(r rune) bool {
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

func (s *StringSliceJSONFormat) Scan(value any) error {
	switch v := value.(type) {
	case string:
		return errors.WithStack(json.Unmarshal([]byte(v), s))
	case []byte:
		return errors.WithStack(json.Unmarshal(v, s))
	default:
		return errors.Errorf("StringSliceJSONFormat.Scan: cannot scan type %T into StringSliceJSONFormat", value)
	}
}

func (s StringSliceJSONFormat) Value() (driver.Value, error) {
	b, err := json.Marshal(s)
	return string(b), errors.WithStack(err)
}
