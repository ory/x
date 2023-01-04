// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package stringsx

func DefaultIfEmpty(s string, defaultValue string) string {
	if len(s) == 0 {
		return defaultValue
	}
	return s
}
