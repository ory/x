package stringsx

import "unicode/utf8"

// TruncateByLength returns string truncated at the end with the length specified
func TruncateByLength(s string, length int) string {
	if length > 0 && len(s) > length {
		res := s[:length]

		// in case we cut in the middle of an utf8 rune, we have to remove the last byte as well until it fits
		for !utf8.ValidString(res) {
			res = res[:len(res)-1]
		}
		return res
	}
	return s
}
