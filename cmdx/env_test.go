// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package cmdx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvVarExamplesHelpMessage(t *testing.T) {
	assert.NotEmpty(t, EnvVarExamplesHelpMessage(""))
}
