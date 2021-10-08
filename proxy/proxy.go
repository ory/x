package proxy

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"github.com/tidwall/sjson"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/ory/graceful"
	"github.com/ory/herodot"
	"github.com/ory/x/httpx"
	"github.com/ory/x/logrusx"
	"github.com/ory/x/urlx"
	"github.com/pkg/errors"
	"github.com/square/go-jose/v3"
	"github.com/tidwall/gjson"
	"github.com/urfave/negroni"
)

const originalHostHeader = "Ory-Internal-Original-Host"

type (
	options struct {
		l          *logrusx.Logger
		hostMapper func(string) *HostConfig
		onError    func(*http.Response, error)
		writer     *herodot.JSONWriter
	}
	Proxy interface {
		HeaderRewrite(*options) error
		BodyRewrite(*options) error
	}
	Request    http.Request
	Response   http.Response
	HostConfig struct {
		CookieHost   string
		UpstreamHost string
		OriginalHost string
	}
	Options func(*options)
)

func WithLogger(l *logrusx.Logger) Options {
	return func(o *options) {
		o.l = l
	}
}

func WithHostMapper(hm func(host string) *HostConfig) Options {
	return func(o *options) {
		o.hostMapper = hm
	}
}

func WithOnError(onErr func(*http.Response, error)) Options {
	return func(o *options) {
		o.onError = onErr
	}
}

func (req *Request) HeaderRewrite(o *options) error {
	c := o.hostMapper(req.URL.Host)
	req.Header.Set(originalHostHeader, req.URL.Host)
	req.URL.Host = c.UpstreamHost

	enc := url.Values{ // TODO maybe replace with JSON
		"cookie_host": []string{},
	}.Encode()

	return nil
}

func (req *Request) ToHttpRequest() *http.Request {
	x := http.Request(*req)
	return &x
}

func (req *Request) BodyRewrite(o *options) error {
	if req.ContentLength == 0 {
		return nil
	}

	c := o.hostMapper(req.URL.Host)

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

func (res *Response) HeaderRewrite(o *options) error {
	host := res.Header.Get(originalHostHeader)
	res.Header.Del(originalHostHeader)
	redir, err := res.ToHttpResponse().Location()
	if err != nil {
		return err
	}
	if redir != nil {
		redir.Host = host
		res.Header.Set("Location", redir.String())
	}

	return nil
}

func (res *Response) BodyRewrite(o *options) error {
	if res.ContentLength == 0 {
		return nil
	}

	redir, err := res.ToHttpResponse().Location()
	if err != nil {
		return err
	}

	c := o.hostMapper(redir.Host)

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return err
		}
		return nil
	}

	switch res.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return err
		}
		defer reader.Close()

		var decoded bytes.Buffer
		if _, err := io.Copy(&decoded, reader); err != nil {
			return err
		}

		body = decoded.Bytes()

		var buf bytes.Buffer
		writer := gzip.NewWriter(&buf)
		if _, err := writer.Write(body); err != nil {
			return err
		}
		if err := writer.Close(); err != nil {
			return err
		}

		res.Body = ioutil.NopCloser(bytes.NewReader(buf.Bytes()))
	default:
		res.Body = ioutil.NopCloser(bytes.NewReader(body))
	}

	// Modify the Logout URL
	if lo := gjson.GetBytes(body, "logout_url"); lo.Exists() {
		p, err := url.ParseRequestURI(lo.String())
		if err != nil {
			return err
		}
		p.Host = c.OriginalHost
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
		p.Host = c.OriginalHost
		body, err = sjson.SetBytes(body, "ui.action", p.String())
		if err != nil {
			return err
		}
	}

	return nil
}

func (res *Response) ToHttpResponse() *http.Response {
	x := http.Response(*res)
	return &x
}

func director(o *options) func(*http.Request) {
	return func(r *http.Request) {
		req := Request(*r)
		err := req.HeaderRewrite(o)
		if err != nil {
			o.onError(req.Response, err)
			return
		}
		err = req.BodyRewrite(o)
		if err != nil {
			o.onError(req.Response, err)
		}
		r = req.ToHttpRequest()
	}
}

func modifyResponse(o *options) func(*http.Response) error {
	return func(r *http.Response) error {
		res := Response(*r)
		err := res.HeaderRewrite(o)
		if err != nil {
			o.onError(r, err)
		}
		err = res.BodyRewrite(o)
		if err != nil {
			o.onError(r, err)
		}
		r = res.ToHttpResponse()
		return nil
	}
}

func New(opts ...Options) *http.Server {
	o := &options{}

	for _, op := range opts {
		op(o)
	}

	handler := &httputil.ReverseProxy{
		Director:       director(o),
		ModifyResponse: modifyResponse(o),
	}

	o.writer = herodot.NewJSONWriter(o.l)

	mw := negroni.New()

	mw.UseHandler(handler)

	server := graceful.WithDefaults(&http.Server{
		Handler: mw,
	})

	return server
}

func checkOry(conf *config, writer herodot.Writer, l *logrusx.Logger, keys *jose.JSONWebKeySet, sig jose.Signer, endpoint *url.URL) func(http.ResponseWriter, *http.Request, http.HandlerFunc) {
	hc := httpx.NewResilientClient(httpx.ResilientClientWithMaxRetry(5), httpx.ResilientClientWithMaxRetryWait(time.Millisecond*5), httpx.ResilientClientWithConnectionTimeout(time.Second*2))


	oryUpstream := httputil.NewSingleHostReverseProxy(endpoint)

	// Did someone say "HACK THE PLANET"? Or rather "HACK THE COOKIES"? Yup...
	oryUpstream.ModifyResponse = func(res *http.Response) error {
		if !strings.EqualFold(res.Request.Host, endpoint.Host) {
			// not ory
			return nil
		}


		body, err := readBody(res)
		if err != nil {
			return err
		} else if body == nil {
			return nil
		}

		return writeBody(res, body)
	}

	return func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		if r.URL.Path == "/.ory/proxy/jwks.json" {
			writer.Write(w, r, publicKeys)
			return
		}

		switch r.URL.Path {
		case "/.ory/jwks.json":
			writer.Write(w, r, publicKeys)
			return
		case "/.ory/init/login":
			http.Redirect(w, r, initUrl(r, "login", conf), http.StatusSeeOther)
			return
		case "/.ory/init/registration":
			http.Redirect(w, r, initUrl(r, "registration", conf), http.StatusSeeOther)
			return
		case "/.ory/init/recovery":
			http.Redirect(w, r, initUrl(r, "recovery", conf), http.StatusSeeOther)
			return
		case "/.ory/init/verification":
			http.Redirect(w, r, initUrl(r, "verification", conf), http.StatusSeeOther)
			return
		case "/.ory/init/settings":
			http.Redirect(w, r, initUrl(r, "settings", conf), http.StatusSeeOther)
			return
		case "/.ory/api/kratos/public/self-service/logout":
			q := r.URL.Query()
			q.Set("return_to", conf.selfURL.String())
			r.URL.RawQuery = q.Encode()
		}

		// We proxy ory things
		if strings.HasPrefix(r.URL.Path, "/.ory") {
			r.URL.Path = strings.ReplaceAll(r.URL.Path, "/.ory/", "/")
			r.Host = endpoint.Host
			q := r.URL.Query()
			q.Set("isProxy", "true")
			r.URL.RawQuery = q.Encode()

			l.WithRequest(r).
				WithField("forwarding_path", r.URL.String()).
				WithField("forwarding_host", r.Host).
				Debug("Forwarding request to Ory.")
			oryUpstream.ServeHTTP(w, r)
			return
		}

		if conf.noUpstream {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		session, err := checkSession(hc, r, endpoint)
		r.Header.Del("Authorization")
		if err != nil || !gjson.GetBytes(session, "active").Bool() {
			next(w, r)
			return
		}

	}
}

func checkSession(c *retryablehttp.Client, r *http.Request, target *url.URL) (json.RawMessage, error) {
	target = urlx.Copy(target)
	target.Path = filepath.Join(target.Path, "api", "kratos", "public", "sessions", "whoami")
	req, err := retryablehttp.NewRequest("GET", target.String(), nil)
	if err != nil {
		return nil, errors.WithStack(herodot.ErrInternalServerError)
	}

	req.Header.Set("Cookie", r.Header.Get("Cookie"))
	req.Header.Set("Authorization", r.Header.Get("Authorization"))
	req.Header.Set("X-Session-Token", r.Header.Get("X-Session-Token"))
	req.Header.Set("X-Request-Id", r.Header.Get("X-Request-Id"))
	req.Header.Set("Accept", "application/json")

	res, err := c.Do(req)
	if err != nil {
		return nil, errors.WithStack(herodot.ErrInternalServerError.WithReasonf("Unable to call session checker: %s", err).WithWrap(err))
	}
	defer res.Body.Close()

	var body json.RawMessage
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, errors.WithStack(herodot.ErrInternalServerError.WithReasonf("Unable to decode session to JSON: %s", err).WithWrap(err))
	}

	return body, nil
}
