// This is an implementation of a koanf.Provider that reads OAuth2
//client id and client secret from the environment variables
//and overrides respective values in Kratos config.
package configx

import (
	"errors"
	"fmt"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/maps"
	"os"
	"regexp"
	"strings"
)

type OidcSecrets struct {
	ko *koanf.Koanf
}

func OidcSecretsProvider(ko *koanf.Koanf) *OidcSecrets {
	return &OidcSecrets{ko: ko}
}

func (p *OidcSecrets) Read() (map[string]interface{}, error) {
	mp := make(map[string]interface{})

	const providersPath = "selfservice.methods.oidc.config.providers"
	if p.ko.Get(providersPath) == nil {
		return nil, nil
	}
	providers := p.ko.Get(providersPath).([]interface{})

	envPrefix := strings.Replace(providersPath, ".", "_", -1)
	r, _ := regexp.Compile(fmt.Sprintf("(?i)^%s_([a-zA-Z0-9]+)_([a-zA-Z0-9_]+)$", envPrefix))
	// Collect the environment variable keys.
	for _, k := range os.Environ() {
		parts := strings.SplitN(k, "=", 2)
		matches := r.FindStringSubmatch(parts[0])
		if matches == nil {
			continue
		}
		for _, prv := range providers {
			provider := prv.(map[string]interface{})
			if strings.EqualFold(provider["id"].(string), matches[1]) {
				provider[strings.ToLower(matches[2])] = parts[1]
			}
		}
	}

	mp[providersPath] = providers
	return maps.Unflatten(mp, "."), nil
}

// ReadBytes is not supported.
func (p *OidcSecrets) ReadBytes() ([]byte, error) {
	return nil, errors.New("oidc secrets provider does not support this method")
}

// Watch is not supported.
func (p *OidcSecrets) Watch(cb func(event interface{}, err error)) error {
	return errors.New("oidc secrets provider does not support this method")
}
