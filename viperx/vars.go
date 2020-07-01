package viperx

import (
	"bytes"
	"encoding/json"

	"github.com/ory/viper"
	"github.com/pkg/errors"

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
	if err := json.NewEncoder(&b).Encode(mapx.ToJSONMap(viper.Get(key))); err != nil {
		return errors.WithStack(err)
	}
	return errors.WithStack(json.NewDecoder(&b).Decode(destination))
}
