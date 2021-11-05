package pagination

// MaxItemsPerPage is used to prevent DoS attacks against large lists by limiting the items per page to 500.
func MaxItemsPerPage(max, is int) int {
	if is > max {
		return max
	}
	return is
}
