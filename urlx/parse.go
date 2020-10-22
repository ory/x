package urlx

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/ory/x/logrusx"
)

// winPathRegex is a regex for [DRIVE-LETTER]:
var winPathRegex = regexp.MustCompile("^[A-Za-z]:.*")

// Parse parses rawURL into a URL structure with special handling for file:// URLs
// File URLs with relative paths (file://../file, ../file) will be returned as a
// utl.URL object without the Scheme set to "file". This is because the file
// scheme doesn't support relative paths. Make sure to check for
// both "file" or "" (an empty string) in URL.Scheme if you are looking for
// a file path.
// Use the companion function GetURLFilePath() to get a file path suitable
// for the current operaring system.
func Parse(rawURL string) (*url.URL, error) {
	lcRawURL := strings.ToLower(rawURL)
	if strings.HasPrefix(lcRawURL, "file:///") {
		return url.Parse(rawURL)
	}

	if strings.HasPrefix(lcRawURL, "file://") {
		// Normally the first part after file:// is a hostname, but since
		// this is often misused we interpret the URL like a normal path
		// by removing the "file://" from the beginning
		rawURL = rawURL[7:]
	}

	if winPathRegex.MatchString(rawURL) {
		// Windows path
		return url.Parse("file:///" + rawURL)
	}

	if strings.HasPrefix(lcRawURL, "\\\\") {
		// Windows UNC path
		// We extract the hostname and create an appropriate file:// URL
		// based on the hostname and the path
		host, path := extractUNCPathParts(rawURL)
		// It is safe to replace the \ with / here because this is POSIX style path
		return url.Parse("file://" + host + strings.ReplaceAll(path, "\\", "/"))
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// ParseOrPanic parses a url or panics.
func ParseOrPanic(in string) *url.URL {
	out, err := Parse(in)
	if err != nil {
		panic(err.Error())
	}
	return out
}

// ParseOrFatal parses a url or fatals.
func ParseOrFatal(l *logrusx.Logger, in string) *url.URL {
	out, err := Parse(in)
	if err != nil {
		l.WithError(err).Fatalf("Unable to parse url: %s", in)
	}
	return out
}

// ParseRequestURIOrPanic parses a request uri or panics.
func ParseRequestURIOrPanic(in string) *url.URL {
	out, err := url.ParseRequestURI(in)
	if err != nil {
		panic(err.Error())
	}
	return out
}

// ParseRequestURIOrFatal parses a request uri or fatals.
func ParseRequestURIOrFatal(l *logrusx.Logger, in string) *url.URL {
	out, err := url.ParseRequestURI(in)
	if err != nil {
		l.WithError(err).Fatalf("Unable to parse url: %s", in)
	}
	return out
}

func extractUNCPathParts(uncPath string) (host, path string) {
	parts := strings.Split(strings.TrimPrefix(uncPath, "\\\\"), "\\")
	host = parts[0]
	if len(parts) > 0 {
		path = "\\" + strings.Join(parts[1:], "\\")
	}
	return host, path
}
