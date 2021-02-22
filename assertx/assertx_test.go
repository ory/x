package assertx

import "testing"

func TestEqualAsJSONExcept(t *testing.T) {
	a := map[string]interface{}{"foo": "bar", "baz": "bar", "bar": "baz"}
	b := map[string]interface{}{"foo": "bar", "baz": "bar", "bar": "not-baz"}

	EqualAsJSONExcept(t, a, b, []string{"bar"})
}
