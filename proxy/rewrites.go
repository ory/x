package proxy

import (
	"bytes"
	"compress/gzip"
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

func HeaderRequestRewrite(req *http.Request, c *HostConfig, opt *options) error {
	shadow, err := url.Parse(c.ShadowURL)
	if err != nil {
		return err
	}

	upstream, err := url.Parse(c.UpstreamHost)
	if err != nil {
		return err
	}

	req.URL.Scheme = upstream.Scheme
	req.URL.Host = upstream.Host
	req.URL.Path = opt.mutateReqPath(req.URL.Path)

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
		co.Path = opt.mutateReqPath(co.Path)
		req.Header.Add("Set-Cookie", co.String())
	}

	return nil
}

func BodyRequestRewrite(req *http.Request, c *HostConfig, opt *options) ([]byte, error) {
	if req.ContentLength == 0 {
		return nil, nil
	}

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, err
		}
		return nil, err
	}

	originalHost, err := url.Parse(c.OriginalHost)
	if err != nil {
		return nil, err
	}

	shadowHost, err := url.Parse(c.ShadowURL)
	if err != nil {
		return nil, err
	}

	body, err = rewriteJson(opt, body, originalHost.Host, shadowHost.Host)
	return body, err
}

func HeaderResponseRewrite(resp *http.Response, c *HostConfig, opt *options) error {
	original, err := url.Parse(c.OriginalHost)
	if err != nil {
		return err
	}

	// ignore the location error when not present
	redir, err := resp.Location()
	if err == nil {
		redir.Scheme = original.Scheme
		redir.Host = original.Host
		redir.Path = opt.mutateResPath(redir.Path)
		resp.Header.Set("Location", redir.String())
	} else if !errors.Is(err, http.ErrNoLocation) {
		return err
	}

	cookies := resp.Cookies()
	resp.Header.Del("Set-Cookie")
	for _, co := range cookies {
		// only alter cookies that were set by the upstream host for our original host (the proxy's domain)
		if !strings.EqualFold(co.Domain, original.Host) {
			continue
		}
		co.Domain = c.CookieHost
		resp.Header.Add("Set-Cookie", co.String())
		co.Path = opt.mutateResPath(co.Path)
	}

	return nil
}

func BodyResponseRewrite(resp *http.Response, c *HostConfig, opt *options) ([]byte, error) {
	if resp.ContentLength == 0 {
		return nil, nil
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

	shadowHost, err := url.Parse(c.ShadowURL)
	if err != nil {
		return nil, err
	}

	return rewriteJson(opt, body, shadowHost.Host, originalHostP.Host)
}

func rewriteJson(opt *options, body []byte, searchHost, targetHost string) ([]byte, error) {
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