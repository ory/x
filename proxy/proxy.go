package proxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httputil"
)

type (
	RespMiddleware func(resp *http.Response, config *HostConfig, body []byte) ([]byte, error)
	ReqMiddleware  func(req *http.Request, config *HostConfig, body []byte) ([]byte, error)
	HostMapper     func(ctx context.Context, r *http.Request) (*HostConfig, error)
	options        struct {
		hostMapper      HostMapper
		onResError      func(*http.Response, error) error
		onReqError      func(*http.Request, error)
		respMiddlewares []RespMiddleware
		reqMiddlewares  []ReqMiddleware
		transport       http.RoundTripper
	}
	HostConfig struct {
		// CookieDomain is the host under which cookies are set.
		// If left empty, no cookie domain will be set
		CookieDomain string
		// UpstreamHost is the next upstream host the proxy will pass the request to.
		// e.g. fluffy-bear-afiu23iaysd.oryapis.com
		UpstreamHost string
		// UpstreamScheme is the protocol used by the upstream service.
		UpstreamScheme string
		// TargetHost is the final target of the request. Should be the same as UpstreamHost
		// if the request is directly passed to the target service.
		TargetHost string
		// TargetScheme is the final target's scheme
		// (i.e. the scheme the target thinks it is running under)
		TargetScheme string
		// PathPrefix is a prefix that is prepended on the original host,
		// but removed before forwarding.
		PathPrefix string
		// originalHost the original hostname the request is coming from.
		// This value will be maintained internally by the proxy.
		originalHost string
		// originalScheme is the original scheme of the request.
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
		c, err := o.hostMapper(r.Context(), r)
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
		if forwardedHost := r.Header.Get("X-Forwarded-Host"); forwardedHost != "" {
			c.originalHost = forwardedHost
		} else {
			c.originalHost = r.Host
		}

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
			if body, err = m(r, c, body); err != nil {
				o.onReqError(r, err)
				return
			}
		}

		n, err := cb.Write(body)
		if err != nil {
			o.onReqError(r, err)
			return
		}

		r.Header.Del("Content-Length")
		r.ContentLength = int64(n)
		r.Body = io.NopCloser(cb)
	}
}

// modifyResponse is a custom internal function for altering a http.Response
func modifyResponse(o *options) func(*http.Response) error {
	return func(r *http.Response) error {
		var c *HostConfig
		if oh := r.Request.Context().Value(hostConfigKey); oh == nil {
			panic("could not get value from context")
		} else {
			c = oh.(*HostConfig)
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
			if body, err = m(r, c, body); err != nil {
				return o.onResError(r, err)
			}
		}

		n, err := cb.Write(body)
		if err != nil {
			return o.onResError(r, err)
		}

		r.Header.Del("Content-Length")
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
func New(hostMapper HostMapper, opts ...Options) http.Handler {
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
