package cloudx

import (
	"bytes"
	"encoding/json"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"

	"github.com/ory/x/osx"
	"github.com/ory/x/stringsx"
)

func ReadConfigFiles(files []string) ([]json.RawMessage, error) {
	var configs []json.RawMessage
	for _, source := range files {
		config, err := ReadConfigFile(source)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}
	return configs, nil
}

func ReadConfigFile(source string) (json.RawMessage, error) {
	contents, err := osx.ReadFileFromAllSources(source, osx.WithEnabledBase64Loader(), osx.WithEnabledHTTPLoader(), osx.WithEnabledFileLoader())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file: %s", source)
	}

	switch f := stringsx.SwitchExact(filepath.Ext(source)); {
	case f.AddCase(".yaml"):
		fallthrough
	case f.AddCase(".yml"):
		var config json.RawMessage
		if err := yaml.Unmarshal(contents, &config); err != nil {
			return nil, errors.Wrapf(err, "failed to parse YAML file: %s", source)
		}
		return config, nil
	case f.AddCase(".json"):
		var config json.RawMessage
		if err := json.NewDecoder(bytes.NewReader(contents)).Decode(&config); err != nil {
			return nil, errors.Wrapf(err, "failed to parse file `%s` from JSON", source)
		}
		return config, nil
	default:
		return nil, f.ToUnknownCaseErr()
	}
}
