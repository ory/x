// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package stringsx

// Coalesce returns the first non-empty string value
func Coalesce(str ...string) string {
	for _, s := range str {
		if s != "" {
			return s
		}
	}
	return ""
}
