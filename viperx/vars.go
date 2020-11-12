package viperx

import (
	"bytes"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/ory/viper"

	"github.com/ory/x/mapx"
)

// UnmarshalKey unmarshals a viper key into the destination struct. The destination struct
// must be JSON-compatible (i.e. have `json` struct tags)
func UnmarshalKey(key string, destination interface{}) error {
	value := viper.Get(key)
	if value == `null` || value == "" || value == nil {
		value = make(map[string]interface{})
	}

	var b bytes.Buffer

	// This may be a string in the case where a value was provided via an env var.
	// If it's a string, try to decode it as json.
	if v, ok := value.(string); ok {
		if _, err := b.WriteString(v); err != nil {
			return errors.WithStack(err)
		}

		// Try decoding the json directly
		if err := json.NewDecoder(&b).Decode(destination); err == nil {
			return nil
		}
	}

	// If it's not a string or not valid json, use the value as it was originally provided
	if err := json.NewEncoder(&b).Encode(mapx.ToJSONMap(value)); err != nil {
		return errors.WithStack(err)
	}

	return errors.WithStack(json.NewDecoder(&b).Decode(destination))
}
