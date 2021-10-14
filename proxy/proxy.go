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
		hostMapper      func(string) (*HostConfig, error)
		mutateReqPath   func(string) string
		mutateResPath   func(string) string
		onResError      func(*http.Response, error) error
		onReqError      func(*http.Request, error)
		respMiddlewares []RespMiddleware
		reqMiddlewares  []ReqMiddleware
	}
	HostConfig struct {
		// CookieHost the host under which the cookie should be set
		// e.g. example.com
		CookieHost string
		// OriginalHost the original hostname the request is coming from
		// e.g. auth.example.com
		OriginalHost string
		// UpstreamHost the target upstream host the proxy will pass the connection to
		// e.g. fluffy-bear-afiu23iaysd.oryapis.com
		UpstreamHost string
	}
	Options    func(*options)
	contextKey string
)

const hostConfigKey contextKey = "host config"

// director is a custom internal function for altering a http.Request
func director(o *options) func(*http.Request) {
	return func(r *http.Request) {
		c, err := o.hostMapper(r.Host)
		if err != nil {
			o.onReqError(r, err)
			return
		}

		*r = *r.WithContext(context.WithValue(r.Context(), hostConfigKey, c))

		err = HeaderRequestRewrite(r, c)
		if err != nil {
			o.onReqError(r, err)
			return
		}

		body, cb, err := BodyRequestRewrite(r, c, o)
		if err != nil {
			o.onReqError(r, err)
			return
		}

		for _, m := range o.reqMiddlewares {
			if body, err = m(r, body); err != nil {
				o.onReqError(r, err)
				return
			}
		}

		if _, err := cb.Write(body); err != nil {
			o.onReqError(r, err)
			return
		}

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

		err := HeaderResponseRewrite(r, c, o)
		if err != nil {
			return o.onResError(r, err)
		}

		body, cb, err := BodyResponseRewrite(r, c, o)
		if err != nil {
			return o.onResError(r, err)
		}

		for _, m := range o.respMiddlewares {
			if body, err = m(r, body); err != nil {
				return o.onResError(r, err)
			}
		}

		if _, err := cb.Write(body); err != nil {
			return o.onResError(r, err)
		}

		r.Body = io.NopCloser(cb)
		return nil
	}
}

func WithHostMapper(hm func(host string) (*HostConfig, error)) Options {
	return func(o *options) {
		o.hostMapper = hm
	}
}

func WithMutateReqPath(mp func(path string) string) Options {
	return func(o *options) {
		o.mutateReqPath = mp
	}
}

func WithMutateResPath(mp func(path string) string) Options {
	return func(o *options) {
		o.mutateResPath = mp
	}
}

func WithOnError(onErr func(*http.Response, error) error) Options {
	return func(o *options) {
		o.onResError = onErr
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

// New creates a new Proxy
// A Proxy sets up a middleware with custom request and response modification handlers
func New(opts ...Options) http.Handler {
	o := &options{
		mutateResPath: pathNop,
		mutateReqPath: pathNop,
		onReqError:    func(*http.Request, error) {},
		onResError:    func(_ *http.Response, err error) error { return err },
	}

	for _, op := range opts {
		op(o)
	}

	return &httputil.ReverseProxy{
		Director:       director(o),
		ModifyResponse: modifyResponse(o),
	}
}

func pathNop(s string) string {
	return s
}
