package sqlxx

import (
	"database/sql/driver"
	"fmt"
	"github.com/ory/x/stringsx"
	"strings"
)

type StringSlicePipeDelimiter []string

func (n *StringSlicePipeDelimiter) Scan(value interface{}) error {
	*n = scanStringSlice('|', value)
	return nil
}

func (n StringSlicePipeDelimiter) Value() (driver.Value, error) {
	return valueStringSlice('|', n), nil
}

func scanStringSlice(delimiter rune, value interface{}) []string {
	return stringsx.Splitx(fmt.Sprintf("%s", value), string(delimiter))
}

func valueStringSlice(delimiter rune, value []string) string {
	return strings.Join(value, string(delimiter))
}
