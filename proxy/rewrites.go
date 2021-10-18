package proxy

import (
	"bytes"
	"compress/gzip"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"path"
	"strings"
)

type compressableBody struct {
	buf bytes.Buffer
	w   io.WriteCloser
}

func (b *compressableBody) Write(d []byte) (int, error) {
	if b == nil {
		// this happens when the body is empty
		return 0, nil
	}

	var w io.Writer = &b.buf
	if b.w != nil {
		w = b.w
		defer b.w.Close()
	}
	return w.Write(d)
}

func (b *compressableBody) Read(p []byte) (n int, err error) {
	if b == nil {
		// this happens when the body is empty
		return 0, io.EOF
	}
	return b.buf.Read(p)
}

func HeaderRequestRewrite(req *http.Request, c *HostConfig) {
	req.URL.Scheme = c.UpstreamProtocol
	req.URL.Host = c.UpstreamHost
	req.URL.Path = strings.TrimPrefix(req.URL.Path, c.PathPrefix)

	if _, ok := req.Header["User-Agent"]; !ok {
		// explicitly disable User-Agent so it's not set to default value
		req.Header.Set("User-Agent", "")
	}
}

func BodyRequestRewrite(req *http.Request, c *HostConfig) ([]byte, *compressableBody, error) {
	if req.ContentLength == 0 {
		return nil, nil, nil
	}

	body, cb, err := readBody(req.Header, req.Body)
	if err != nil {
		return nil, nil, err
	}

	return bytes.ReplaceAll(body, []byte(c.originalHost+c.PathPrefix), []byte(c.UpstreamHost)), cb, nil
}

func HeaderResponseRewrite(resp *http.Response, c *HostConfig) error {
	redir, err := resp.Location()
	if err != nil {
		if !errors.Is(err, http.ErrNoLocation) {
			return errors.WithStack(err)
		}
	} else {
		redir.Scheme = c.originalScheme
		redir.Host = c.originalHost
		redir.Path = path.Join(c.PathPrefix, redir.Path)
		resp.Header.Set("Location", redir.String())
	}

	cookies := resp.Cookies()
	resp.Header.Del("Set-Cookie")
	for _, co := range cookies {
		// only alter cookies that were set by the upstream host for our original host (the proxy's domain)
		cDomain := stripPort(c.UpstreamHost) // cookies don't distinguish ports
		if strings.EqualFold(co.Domain, cDomain) {
			co.Domain = c.CookieDomain
		}
		resp.Header.Add("Set-Cookie", co.String())
	}

	return nil
}

func BodyResponseRewrite(resp *http.Response, c *HostConfig) ([]byte, *compressableBody, error) {
	if resp.ContentLength == 0 {
		return nil, nil, nil
	}

	body, cb, err := readBody(resp.Header, resp.Body)
	if err != nil {
		return nil, nil, err
	}

	return bytes.ReplaceAll(body, []byte(c.UpstreamHost), []byte(c.originalHost+c.PathPrefix)), cb, nil
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

		cb.w = gzip.NewWriter(&cb.buf)
	default:
		// do nothing, we can read directly
	}

	b, err := io.ReadAll(body)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	return b, cb, nil
}

// stripPort removes the optional port from the host. It does not validate the port or host.
// Supports DNS and IPv4 (but not IPv6) hosts.
func stripPort(host string) string {
	colon := strings.LastIndexByte(host, ':')
	if colon != -1 {
		host = host[:colon]
	}
	return host
}
