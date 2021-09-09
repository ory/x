//go:build !windows
// +build !windows

package urlx

import (
	"net/url"
)

// GetURLFilePath returns the path of a URL that is compatible with the runtime os filesystem
func GetURLFilePath(u *url.URL) string {
	if u == nil {
		return ""
	}
	return u.Path
}
