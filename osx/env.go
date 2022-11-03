// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package osx

import "os"

// GetenvDefault returns an environment variable or the default value if it is empty.
func GetenvDefault(key string, def string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return def
}
