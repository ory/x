package osx

import (
	"encoding/base64"
	"io"
	"net/url"
	"os"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"

	"github.com/ory/x/httpx"
)

type options struct {
	disableFileLoader   bool
	disableHTTPLoader   bool
	disableBase64Loader bool
	base64enc           *base64.Encoding
	hc                  *retryablehttp.Client
}

type Option func(o *options)

func newOptions(opts []Option) *options {
	o := &options{
		disableFileLoader:   false,
		disableHTTPLoader:   false,
		disableBase64Loader: false,
		base64enc:           base64.RawURLEncoding,
		hc:                  httpx.NewResilientClient(),
	}

	for _, f := range opts {
		f(o)
	}

	return o
}

// WithDisableFileLoader disables the file loader.
func WithDisableFileLoader() Option {
	return func(o *options) {
		o.disableFileLoader = true
	}
}

// WithDisableHTTPLoader disables the HTTP loader.
func WithDisableHTTPLoader() Option {
	return func(o *options) {
		o.disableHTTPLoader = true
	}
}

// WithDisableBase64Loader disables the base64 loader.
func WithDisableBase64Loader() Option {
	return func(o *options) {
		o.disableBase64Loader = true
	}
}

// WithBase64Encoding sets the base64 encoding.
func WithBase64Encoding(enc *base64.Encoding) Option {
	return func(o *options) {
		o.base64enc = enc
	}
}

//	WithHTTPClient sets the HTTP client.
func WithHTTPClient(hc *retryablehttp.Client) Option {
	return func(o *options) {
		o.hc = hc
	}
}

// ReadFileFromAllSources reads a file from base64, http, https, and file sources.
//
// Using options, you can disable individual loaders. For example, the following will
// return an error:
//
//		ReadFileFromAllSources("https://foo.bar/baz.txt", WithDisableHTTPLoader())
//
// Possible formats are:
//
// - file:///path/to/file
// - https://host.com/path/to/file
// - http://host.com/path/to/file
// - base64://<base64 encoded string>
//
// For more options, check:
//
// - WithDisableFileLoader
// - WithDisableHTTPLoader
// - WithDisableBase64Loader
// - WithBase64Encoding
// - WithHTTPClient
func ReadFileFromAllSources(source string, opts ...Option) (bytes []byte, err error) {
	o := newOptions(opts)

	parsed, err := url.ParseRequestURI(source)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse URL")
	}

	switch parsed.Scheme {
	case "file":
		if o.disableFileLoader {
			return nil, errors.New("file loader disabled")
		}

		bytes, err = os.ReadFile(parsed.Host + parsed.Path)
		if err != nil {
			return nil, errors.Wrap(err, "unable to read the file")
		}
	case "http", "https":
		if o.disableHTTPLoader {
			return nil, errors.New("http(s) loader disabled")
		}
		resp, err := o.hc.Get(parsed.String())
		if err != nil {
			return nil, errors.Wrap(err, "unable to load remote file")
		}
		defer resp.Body.Close()

		bytes, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrap(err, "unable to read the HTTP response body")
		}
	case "base64":
		if o.disableBase64Loader {
			return nil, errors.New("base64 loader disabled")
		}

		bytes, err = o.base64enc.DecodeString(parsed.Host + parsed.RawPath)
		if err != nil {
			return nil, errors.Wrap(err, "unable to base64 decode the location")
		}
	default:
		return nil, errors.Errorf("unsupported source `%s`", parsed.Scheme)
	}

	return bytes, nil

}
