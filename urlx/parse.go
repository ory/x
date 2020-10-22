package urlx

import (
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ory/x/logrusx"
)

// winPathRegex is a regex for [DRIVE-LETTER]:
var winPathRegex = regexp.MustCompile("^[A-Za-z]:.*")

// Parse parses rawurl into a URL structure with special handling for file:// URLs
// File URLs with relative paths (file://../file, ../file) will be returned as a
// utl.URL object without the Scheme set to "file". This is because the file
// scheme doesn't support relative paths. Make sure to check for
// both "file" or "" (an empty string) in URL.Scheme if you are looking for
// a file path.
// Use the companion function GetURLFilePath() to get a file path suitable
// for the current operaring system.
func Parse(rawurl string) (*url.URL, error) {
	lcRawurl := strings.ToLower(rawurl)
	if strings.Index(lcRawurl, "file:///") == 0 {
		return url.Parse("file:///" + toSlash(rawurl[8:]))
	}
	if strings.Index(lcRawurl, "file://") == 0 {
		// Normally the first part after file:// is a hostname, but since
		// this is often misused we interpret the URL like a normal path
		// by removing the "file://" from the beginning
		rawurl = rawurl[7:]
	}
	if winPathRegex.MatchString(rawurl) {
		// Windows path
		return url.Parse("file:///" + toSlash(rawurl))
	}
	if strings.Index(lcRawurl, "\\\\") == 0 {
		// Windows UNC path
		// We extract the hostname and creates an appropriate file:// URL
		// based on the hostname and the path
		parts := strings.Split(filepath.FromSlash(rawurl), "\\")
		host := ""
		if len(parts) > 2 {
			host = parts[2]
		}
		p := "/"
		if len(parts) > 4 {
			p += strings.Join(parts[3:], "/")
		}
		return url.Parse("file://" + host + p)
	}
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "file" || u.Scheme == "" {
		u.Path = toSlash(u.Path)
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

func toSlash(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}
