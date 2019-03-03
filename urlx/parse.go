package urlx

import (
	"github.com/sirupsen/logrus"
	"net/url"
)

// ParseOrPanic parses a url or panics.
func ParseOrPanic(in string) *url.URL {
	out, err := url.Parse(in)
	if err != nil {
		panic(err)
	}
	return out
}

// ParseOrPanic parses a url or fatals.
func ParseOrFatal(l logrus.FieldLogger, in string) *url.URL {
	out, err := url.Parse(in)
	if err != nil {
		l.WithError(err).Fatal("Unable to parse url: %s", in)
	}
	return out
}

// ParseOrPanic parses a request uri or panics.
func ParseRequestURIOrPanic(in string) *url.URL {
	out, err := url.ParseRequestURI(in)
	if err != nil {
		panic(err)
	}
	return out
}

// ParseOrPanic parses a request uri or fatals.
func ParseRequestURIOrFatal(l logrus.FieldLogger, in string) *url.URL {
	out, err := url.ParseRequestURI(in)
	if err != nil {
		l.WithError(err).Fatal("Unable to parse url: %s", in)
	}
	return out
}
