package proxy

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

const originalHostKey = "Ory-Internal-Host-Key"

func HeaderRequestRewrite(req *http.Request, opt *options) (*http.Request, error) {
	c, err := opt.hostMapper(req.Host)
	if err != nil {
		return req, err
	}

	ctx := context.WithValue(req.Context(), originalHostKey, req.Host)
	req = req.WithContext(ctx)

	shadow, err := url.Parse(c.ShadowHost)
	if err != nil {
		return req, err
	}

	upstream, err := url.Parse(c.UpstreamHost)
	if err != nil {
		return req, err
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

	cookies := req.Cookies()
	req.Header.Del("Set-Cookie")
	for _, co := range cookies {
		// only alter cookies that were specifically configured to be set
		if !strings.EqualFold(c.CookieHost, co.Domain) {
			continue
		}
		co.Domain = shadow.Host
		req.Header.Add("Set-Cookie", co.String())
	}

	return req, nil
}

func BodyRequestRewrite(req *http.Request, opt *options) (*http.Request, []byte, error) {
	if req.ContentLength == 0 {
		return req, nil, nil
	}

	c, err := opt.hostMapper(req.URL.Host)

	if err != nil {
		return req, nil, err
	}

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return req, nil, err
		}
		return req, nil, err
	}

	originalHost, err := url.Parse(c.OriginalHost)
	if err != nil {
		return req, nil, err
	}

	shadowHost, err := url.Parse(c.ShadowHost)
	if err != nil {
		return req, nil, err
	}

	body, err = rewriteJson(body, originalHost.Host, shadowHost.Host)
	return req, body, err
}

func HeaderResponseRewrite(resp *http.Response, opt *options) error {
	var originalHost string

	if oh := resp.Request.Context().Value(originalHostKey); oh != nil {
		originalHost = oh.(string)
	}

	c, err := opt.hostMapper(originalHost)

	if err != nil {
		return err
	}

	shadow, err := url.Parse(c.ShadowHost)
	if err != nil {
		return err
	}

	// ignore the location error when not present
	redir, _ := resp.Location()
	if redir != nil {
		redir.Host = opt.server.Addr
		if opt.mutateResPath != nil {
			redir.Path = opt.mutateResPath(redir.Path)
		}
		resp.Header.Set("Location", redir.String())
	}

	cookies := resp.Cookies()
	resp.Header.Del("Set-Cookie")
	for _, co := range cookies {
		// only alter cookies that were set by the upstream host for our shadow host (the proxy's domain)
		if !strings.EqualFold(co.Domain, shadow.Host) {
			continue
		}
		co.Domain = c.CookieHost
		resp.Header.Add("Set-Cookie", co.String())
	}

	return nil
}

func BodyResponseRewrite(resp *http.Response, opt *options) ([]byte, error) {
	if resp.ContentLength == 0 {
		return nil, nil
	}

	var originalHost string

	if oh := resp.Request.Context().Value(originalHostKey); oh != nil {
		originalHost = oh.(string)
	}

	c, err := opt.hostMapper(originalHost)
	if err != nil {
		return nil, err
	}

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

	originalHostP, err := url.Parse(c.OriginalHost)
	if err != nil {
		return nil, err
	}

	shadowHost, err := url.Parse(c.ShadowHost)
	if err != nil {
		return nil, err
	}

	return rewriteJson(body, shadowHost.Host, originalHostP.Host)
}

func rewriteJson(body []byte, searchHost, targetHost string) ([]byte, error) {
	gjson.AddModifier("domain", func(json, arg string) string {
		// if the json contains the argument host
		// replace it with the targethost
		if strings.Contains(json, arg) {
			return strings.Replace(json, arg, targetHost, -1)
		}
		return json
	})

	if s := gjson.GetBytes(body, fmt.Sprintf("*.@domain:%s", searchHost)); s.Exists() {
		body, err := sjson.SetBytes(body, fmt.Sprintf("*.@domain:%s", searchHost), s.String())
		if err != nil {
			return body, err
		}
	}

	return body, nil
}