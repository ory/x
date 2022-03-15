package cloudx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthLogout(t *testing.T) {
	configDir := newConfigDir(t)
	registerAccount(t, configDir)

	exec := configAwareCmd(configDir)
	_, _, err := exec.Exec(nil, "auth", "logout")
	require.NoError(t, err)

	ac := readConfig(t, configDir)
	assert.Empty(t, ac.SessionToken)
}
