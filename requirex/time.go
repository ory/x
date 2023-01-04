// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package requirex

import (
	"fmt"
	"time"

	"github.com/stretchr/testify/require"
)

// EqualDuration fails if expected and actual are more distant than precision
func EqualDuration(t require.TestingT, expected time.Duration, actual time.Duration, precision time.Duration) {
	delta := expected - actual
	if delta < 0 {
		delta = -delta
	}
	require.Less(t, delta, precision, fmt.Sprintf("expected %s; got %s", expected, actual))
}

// EqualTime fails if expected and actual are more distant than precision
func EqualTime(t require.TestingT, expected time.Time, actual time.Time, precision time.Duration) {
	delta := expected.Sub(actual)
	if delta < 0 {
		delta = -delta
	}
	require.Less(t, delta, precision, fmt.Sprintf(
		"expected %s; got %s",
		expected.Format(time.RFC3339Nano),
		actual.Format(time.RFC3339Nano),
	))
}
