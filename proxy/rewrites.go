package proxy

import (
	"bytes"
	"compress/gzip"
	"errors"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

const originalHostHeader = "Ory-Internal-Original-Host"

func urlHostName(hostUrl string) (string, error) {
	u, err := url.Parse(hostUrl)
	if err != nil {
		return "", err
	}
	return u.Hostname(), nil
}

func HeaderRequestRewrite(req *http.Request, opt *options) error {
	c := opt.hostMapper(req.Host)
	req.Header.Set(originalHostHeader, req.Host)

	shadow, err := url.Parse(c.ShadowHost)
	if err != nil {
		return err
	}

	req.Host = shadow.Host

	upstream, err := url.Parse(c.UpstreamHost)
	if err != nil {
		return err
	}

	req.URL.Scheme = upstream.Scheme
	req.URL.Host = upstream.Host

	if opt.mutateReqPath != nil {
		req.URL.Path = opt.mutateReqPath(req.URL.Path)
	}

	targetQuery := upstream.RawQuery

	if targetQuery == "" || req.URL.RawQuery == "" {
		req.URL.RawQuery = targetQuery + req.URL.RawQuery
	} else {
		req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
	}

	if _, ok := req.Header["User-Agent"]; !ok {
		// explicitly disable User-Agent so it's not set to default value
		req.Header.Set("User-Agent", "")
	}

	// TODO maybe replace with JSON
	/*enc := url.Values{
		"cookie_host": []string{},
	}.Encode()*/

	return nil
}

func BodyRequestRewrite(req *http.Request, opt *options) error {
	if req.ContentLength == 0 {
		return nil
	}

	c := opt.hostMapper(req.URL.Host)

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return err
		}
		return nil
	}

	// Modify the Logout URL
	if lo := gjson.GetBytes(body, "logout_url"); lo.Exists() {
		p, err := url.ParseRequestURI(lo.String())
		if err != nil {
			return err
		}
		p.Host = c.UpstreamHost
		body, err = sjson.SetBytes(body, "logout_url", p.String())
		if err != nil {
			return err
		}
	}

	// Modify flow URLs
	if lo := gjson.GetBytes(body, "ui.action"); lo.Exists() {
		p, err := url.ParseRequestURI(lo.String())
		if err != nil {
			return err
		}
		p.Host = c.UpstreamHost
		body, err = sjson.SetBytes(body, "ui.action", p.String())
		if err != nil {
			return err
		}
	}

	return nil
}

func HeaderResponseRewrite(resp *http.Response, opt *options) error {
	upstreamHost := resp.Request.Host
	originalHost := resp.Header.Get(originalHostHeader)
	resp.Header.Del(originalHostHeader)

	// ignore the location error when not present
	redir, _ := resp.Location()
	if redir != nil {
		redir.Host = originalHost
		if opt.mutateResPath != nil {
			redir.Path = opt.mutateResPath(redir.Path)
		}
		resp.Header.Set("Location", redir.String())
	}

	cookies := resp.Cookies()
	resp.Header.Del("Set-Cookie")
	for _, c := range cookies {
		// only alter cookies that were set by the upstream host
		if !strings.EqualFold(c.Domain, upstreamHost) {
			continue
		}
		c.Domain = originalHost
		resp.Header.Add("Set-Cookie", c.String())
	}

	return nil
}

func BodyResponseRewrite(resp *http.Response, opt *options) ([]byte, error) {
	if resp.ContentLength == 0 {
		return nil, nil
	}

	redir, err := resp.Location()
	if err != nil {
		return nil, err
	}

	c := opt.hostMapper(redir.Host)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, err
		}
		return nil, nil
	}

	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		defer reader.Close()

		var decoded bytes.Buffer
		if _, err := io.Copy(&decoded, reader); err != nil {
			return nil, err
		}

		body = decoded.Bytes()

		var buf bytes.Buffer
		writer := gzip.NewWriter(&buf)
		if _, err := writer.Write(body); err != nil {
			return nil, err
		}
		if err := writer.Close(); err != nil {
			return nil, err
		}

		resp.Body = ioutil.NopCloser(bytes.NewReader(buf.Bytes()))
	default:
		resp.Body = ioutil.NopCloser(bytes.NewReader(body))
	}

	// Modify the Logout URL
	if lo := gjson.GetBytes(body, "logout_url"); lo.Exists() {
		p, err := url.ParseRequestURI(lo.String())
		if err != nil {
			return nil, err
		}
		p.Host = c.OriginalHost
		body, err = sjson.SetBytes(body, "logout_url", p.String())
		if err != nil {
			return nil, err
		}
	}

	// Modify flow URLs
	if lo := gjson.GetBytes(body, "ui.action"); lo.Exists() {
		p, err := url.ParseRequestURI(lo.String())
		if err != nil {
			return nil, err
		}
		p.Host = c.OriginalHost
		body, err = sjson.SetBytes(body, "ui.action", p.String())
		if err != nil {
			return nil, err
		}
	}

	return body, nil
}
