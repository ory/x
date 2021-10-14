package proxy

import (
	"bytes"
	"compress/gzip"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"regexp"
	"strings"
)

type compressableBody struct {
	bytes.Buffer
	io.Writer
}

func (b *compressableBody) Write(d []byte) (int, error) {
	if b == nil {
		// this happens when the body is empty
		return 0, nil
	}

	var w io.Writer = &b.Buffer
	if b.Writer != nil {
		w = b.Writer
	}
	return w.Write(d)
}

func HeaderRequestRewrite(req *http.Request, c *HostConfig) error {
	req.URL.Host = c.UpstreamHost

	if _, ok := req.Header["User-Agent"]; !ok {
		// explicitly disable User-Agent so it's not set to default value
		req.Header.Set("User-Agent", "")
	}

	return nil
}

func BodyRequestRewrite(req *http.Request, c *HostConfig, o *options) ([]byte, *compressableBody, error) {
	if req.ContentLength == 0 {
		return nil, nil, nil
	}

	body, cb, err := readBody(req.Header, req.Body)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	r, err := hostAndPathURLRegexp(c.OriginalHost)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	return r.ReplaceAllFunc(body, replaceURLs(c.OriginalHost, c.UpstreamHost, o.mutateReqPath)), cb, nil
}

func HeaderResponseRewrite(resp *http.Response, c *HostConfig, o *options) error {
	redir, err := resp.Location()
	if err != nil && !errors.Is(err, http.ErrNoLocation) {
		return errors.WithStack(err)
	} else {
		redir.Host = c.OriginalHost
		redir.Path = o.mutateResPath(redir.Path)
		resp.Header.Set("Location", redir.String())
	}

	cookies := resp.Cookies()
	resp.Header.Del("Set-Cookie")
	for _, co := range cookies {
		// only alter cookies that were set by the upstream host for our original host (the proxy's domain)
		if !strings.EqualFold(co.Domain, c.UpstreamHost) {
			continue
		}
		co.Domain = c.CookieHost
		resp.Header.Add("Set-Cookie", co.String())
	}

	return nil
}

func BodyResponseRewrite(resp *http.Response, c *HostConfig, o *options) ([]byte, *compressableBody, error) {
	if resp.ContentLength == 0 {
		return nil, nil, nil
	}

	body, cb, err := readBody(resp.Header, resp.Body)

	r, err := hostAndPathURLRegexp(c.UpstreamHost)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	return r.ReplaceAllFunc(body, replaceURLs(c.UpstreamHost, c.OriginalHost, o.mutateResPath)), cb, nil
}

func readBody(h http.Header, body io.ReadCloser) ([]byte, *compressableBody, error) {
	defer body.Close()

	cb := &compressableBody{}

	switch h.Get("Content-Encoding") {
	case "gzip":
		var err error
		body, err = gzip.NewReader(body)
		if err != nil {
			return nil, nil, errors.WithStack(err)
		}

		cb.Writer = gzip.NewWriter(&cb.Buffer)
	default:
		// do nothing, we can read directly
	}

	b, err := io.ReadAll(body)
	return b, cb, err
}

// The path chars are taken from https://datatracker.ietf.org/doc/html/rfc3986#section-3.3
//
// > The path is terminated by the first question mark ("?") or number sign ("#") character, or
// > by the end of the URI.
const pathPattern = "[a-zA-Z0-9\\-._~!$&'()*+,;=:@]*"

func hostAndPathURLRegexp(host string) (*regexp.Regexp, error) {
	return regexp.Compile("\\Q" + host + "\\E" + pathPattern)
}

func replaceURLs(oldHost, newHost string, mutatePath func(string) string) func([]byte) []byte {
	return func(match []byte) []byte {
		path := strings.TrimPrefix(string(match), oldHost)
		return []byte(newHost + mutatePath(path))
	}
}
