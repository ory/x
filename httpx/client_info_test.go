package httpx

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIgnoresInternalIPs(t *testing.T) {
	input := "54.155.246.232,10.145.1.10"

	res, err := GetClientIPAddressesWithoutInternalIPs(strings.Split(input, ","))
	require.NoError(t, err)
	assert.Equal(t, "54.155.246.232", res)
}

func TestEmptyInputArray(t *testing.T) {
	res, err := GetClientIPAddressesWithoutInternalIPs([]string{})
	require.NoError(t, err)
	assert.Equal(t, "", res)
}
