package proxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httputil"
)

type (
	RespMiddleware func(resp *http.Response, body []byte) ([]byte, error)
	ReqMiddleware  func(req *http.Request, body []byte) ([]byte, error)
	options        struct {
		hostMapper      func(context.Context, string) (*HostConfig, error)
		onResError      func(*http.Response, error) error
		onReqError      func(*http.Request, error)
		respMiddlewares []RespMiddleware
		reqMiddlewares  []ReqMiddleware
		transport       http.RoundTripper
	}
	HostConfig struct {
		// CookieDomain the host under which the cookie should be set
		// e.g. example.com
		// If left empty, will ask the browser to use the browser address bar's host (default HTTP Cookie behavior)
		CookieDomain string
		// UpstreamHost the target upstream host the proxy will pass the connection to
		// e.g. fluffy-bear-afiu23iaysd.oryapis.com
		UpstreamHost string
		// PathPrefix is a prefix that is prepended on the original host,
		// but removed on the upstream.
		PathPrefix string
		// UpstreamProtocol is the protocol used by the upstream.
		UpstreamProtocol string
		// originalHost the original hostname the request is coming from
		// e.g. auth.example.com
		// This value will be maintained internally by the proxy.
		originalHost string
		// originalScheme is the original scheme of the request
		// This value will be maintained internally by the proxy.
		originalScheme string
	}
	Options    func(*options)
	contextKey string
)

const (
	hostConfigKey contextKey = "host config"
)

// director is a custom internal function for altering a http.Request
func director(o *options) func(*http.Request) {
	return func(r *http.Request) {
		c, err := o.hostMapper(r.Context(), r.Host)
		if err != nil {
			o.onReqError(r, err)
			return
		}

		if forwardedProto := r.Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
			c.originalScheme = forwardedProto
		} else if r.TLS == nil {
			c.originalScheme = "http"
		} else {
			c.originalScheme = "https"
		}

		c.originalHost = r.Host
		*r = *r.WithContext(context.WithValue(r.Context(), hostConfigKey, c))

		headerRequestRewrite(r, c)

		var body []byte
		var cb *compressableBody

		if r.ContentLength != 0 {
			body, cb, err = readBody(r.Header, r.Body)
			if err != nil {
				o.onReqError(r, err)
				return
			}
		}

		for _, m := range o.reqMiddlewares {
			if body, err = m(r, body); err != nil {
				o.onReqError(r, err)
				return
			}
		}

		n, err := cb.Write(body)
		if err != nil {
			o.onReqError(r, err)
			return
		}

		r.ContentLength = int64(n)
		r.Body = io.NopCloser(cb)
	}
}

// modifyResponse is a custom internal function for altering a http.Response
func modifyResponse(o *options) func(*http.Response) error {
	return func(r *http.Response) error {
		var c *HostConfig
		if oh := r.Request.Context().Value(hostConfigKey); oh != nil {
			c = oh.(*HostConfig)
		} else {
			panic("could not get value from context")
		}

		err := headerResponseRewrite(r, c)
		if err != nil {
			return o.onResError(r, err)
		}

		body, cb, err := bodyResponseRewrite(r, c)
		if err != nil {
			return o.onResError(r, err)
		}

		for _, m := range o.respMiddlewares {
			if body, err = m(r, body); err != nil {
				return o.onResError(r, err)
			}
		}

		n, err := cb.Write(body)
		if err != nil {
			return o.onResError(r, err)
		}

		r.ContentLength = int64(n)
		r.Body = io.NopCloser(cb)
		return nil
	}
}

func WithOnError(onReqErr func(*http.Request, error), onResErr func(*http.Response, error) error) Options {
	return func(o *options) {
		o.onReqError = onReqErr
		o.onResError = onResErr
	}
}

func WithReqMiddleware(middlewares ...ReqMiddleware) Options {
	return func(o *options) {
		o.reqMiddlewares = append(o.reqMiddlewares, middlewares...)
	}
}

func WithRespMiddleware(middlewares ...RespMiddleware) Options {
	return func(o *options) {
		o.respMiddlewares = append(o.respMiddlewares, middlewares...)
	}
}

func WithTransport(t http.RoundTripper) Options {
	return func(o *options) {
		o.transport = t
	}
}

// New creates a new Proxy
// A Proxy sets up a middleware with custom request and response modification handlers
func New(hostMapper func(ctx context.Context, host string) (*HostConfig, error), opts ...Options) http.Handler {
	o := &options{
		hostMapper: hostMapper,
		onReqError: func(*http.Request, error) {},
		onResError: func(_ *http.Response, err error) error { return err },
		transport:  http.DefaultTransport,
	}

	for _, op := range opts {
		op(o)
	}

	return &httputil.ReverseProxy{
		Director:       director(o),
		ModifyResponse: modifyResponse(o),
		Transport:      o.transport,
	}
}
