// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package stringslice

// Unique returns the given string slice with unique values.
func Unique(i []string) []string {
	u := make([]string, 0, len(i))
	m := make(map[string]bool)

	for _, val := range i {
		if _, ok := m[val]; !ok {
			m[val] = true
			u = append(u, val)
		}
	}

	return u
}
