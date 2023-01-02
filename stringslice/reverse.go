// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package stringslice

func Reverse(s []string) []string {
	r := make([]string, len(s))

	for i, j := 0, len(r)-1; i <= j; i, j = i+1, j-1 {
		r[i], r[j] = s[j], s[i]
	}

	return r
}
