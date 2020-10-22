package urlx

import (
	"net/url"
	"path/filepath"
	"runtime"
)

// GetURLFilePath returns the path of a URL that is compatible with the runtime os filesystem
func GetURLFilePath(u *url.URL) string {
	if u == nil {
		return ""
	}

	if u.Scheme != "file" && u.Scheme != "" {
		return u.Path
	}
	fPath := u.Path
	if runtime.GOOS == "windows" {
		if u.Host != "" {
			// Make UNC Path
			s := string(filepath.Separator)
			fPath = s + s + u.Host + filepath.FromSlash(fPath)
			return fPath
		}
		if winPathRegex.MatchString(fPath[1:]) {
			// On Windows we should remove the initial path separator in case this
			// is a normal path (for example: "\c:\" -> "c:\"")
			fPath = stripFistPathSeparators(fPath)
		}
	}
	return filepath.FromSlash(fPath)
}

func stripFistPathSeparators(fPath string) string {
	for len(fPath) > 0 && (fPath[0] == '/' || fPath[0] == '\\') {
		fPath = fPath[1:]
	}
	return fPath
}
