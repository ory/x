// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package contextx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTreeContext(t *testing.T) {
	assert.True(t, IsRootContext(RootContext))
	assert.True(t, IsRootContext(context.WithValue(RootContext, "foo", "bar")))
	assert.False(t, IsRootContext(context.Background()))
}
