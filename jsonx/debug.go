package jsonx

import (
	"encoding/json"
	"fmt"
)

// Anonymize takes a JSON byte array and anonymizes its content by
// recursively replacing all values with a string indicating their type.
//
// It recurses into nested objects and arrays, but ignores the "schemas" and "id".
func Anonymize(data []byte, except ...string) []byte {
	obj := make(map[string]any)
	if err := json.Unmarshal(data, &obj); err != nil {
		return []byte(fmt.Sprintf(`{"error": "invalid JSON", "message": %q}`, err.Error()))
	}

	anonymize(obj, except...)

	out, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return []byte(fmt.Sprintf(`{"error": "could not marshal JSON shape", "message": %q}`, err.Error()))
	}

	return out
}

func anonymize(obj map[string]any, except ...string) {
	for k, v := range obj {
		if k == "schemas" || k == "id" {
			continue
		}

		switch v := v.(type) {
		case []any:
			for elIdx, el := range v {
				switch el := el.(type) {
				case map[string]any:
					anonymize(el)
					v[elIdx] = el
				default:
					v[elIdx] = fmt.Sprintf("%T", el)
				}
			}

		case map[string]any:
			anonymize(v)
			obj[k] = v
		default:
			obj[k] = fmt.Sprintf("%T", v)
		}
	}
}
