// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package stringslice

// Merge merges several string slices into one.
func Merge(parts ...[]string) []string {
	var result []string
	for _, part := range parts {
		result = append(result, part...)
	}

	return result
}
