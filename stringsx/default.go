// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package stringsx

// Deprecated: use Coalesce instead
func DefaultIfEmpty(s string, defaultValue string) string {
	return Coalesce(s, defaultValue)
}
