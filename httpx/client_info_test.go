package httpx

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestIgnoresInternalIPs(t *testing.T) {
	input := "54.155.246.232,10.145.1.10"

	res, err := GetClientIPAddress(strings.Split(input, ","), InternalIPSet)
	require.NoError(t, err)
	assert.Equal(t, "54.155.246.232", res)
}

func TestEmptyInputArray(t *testing.T) {
	res, err := GetClientIPAddress([]string{}, InternalIPSet)
	require.NoError(t, err)
	assert.Equal(t, "", res)
}
