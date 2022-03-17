package stringsx

func DefaultIfEmpty(s string, defaultValue string) string {
	if len(s) == 0 {
		return defaultValue
	}
	return s
}
