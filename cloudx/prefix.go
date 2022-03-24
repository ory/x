package cloudx

import (
	"encoding/json"

	"github.com/tidwall/sjson"
)

func prefixIdentityConfig(s []string) []string {
	for k := range s {
		s[k] = "/services/identity/config" + s[k]
	}
	return s
}

func prefixFileIdentityConfig(configs []json.RawMessage) ([]json.RawMessage, error) {
	for k := range configs {
		raw, err := sjson.SetRawBytes(json.RawMessage("{}"), "services.identity.config", configs[k])
		if err != nil {
			return nil, err
		}
		configs[k] = raw
	}
	return configs, nil
}
