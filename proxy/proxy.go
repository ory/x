package proxy

import (
	"context"
	"net/http"
	"net/http/httputil"
)

type (
	RespMiddleware    func(resp *http.Response, body []byte) ([]byte, error)
	ReqMiddleware     func(req *http.Request, body []byte) ([]byte, error)
	NegroniMiddleware func(w http.ResponseWriter, r *http.Request, n http.HandlerFunc)
	options           struct {
		hostMapper        func(string) (*HostConfig, error)
		mutateReqPath     func(string) string
		mutateResPath     func(string) string
		onError           func(*http.Response, error) error // todo req & resp separate
		respMiddlewares   []RespMiddleware
		reqMiddlewares    []ReqMiddleware
		handler           *httputil.ReverseProxy
	}
	HostConfig struct {
		// CookieHost the host under which the cookie should be set
		// e.g. example.com
		CookieHost string
		// OriginalHost the original hostname the request is coming from
		// e.g. auth.example.com
		OriginalHost string
		// UpstreamHost the target upstream host the proxy will pass the connection to
		UpstreamHost string
		// ShadowURL the host the proxy is imitating
		ShadowURL string
	}
	Options func(*options)
)

// director is a custom internal function for altering a http.Request
func director(o *options) func(*http.Request) {
	return func(r *http.Request) {
		c, err := o.hostMapper(r.Host)
		if err != nil {
			o.onError(r.Response, err)
			return
		}

		*r = *r.WithContext(context.WithValue(r.Context(), originalHostKey, c))

		err = HeaderRequestRewrite(r, c, o)
		if err != nil {
			o.onError(r.Response, err)
			return
		}

		body, err := BodyRequestRewrite(r, c, o)
		if err != nil {
			o.onError(r.Response, err)
			return
		}

		for _, m := range o.reqMiddlewares {
			if body, err = m(r, body); err != nil {
				o.onError(r.Response, err)
				return
			}
		}
	}
}

// modifyResponse is a custom internal function for altering a http.Response
func modifyResponse(o *options) func(*http.Response) error {
	return func(r *http.Response) error {
		var c *HostConfig
		if oh := r.Request.Context().Value(originalHostKey); oh != nil {
			c = oh.(*HostConfig)
		} else {
			panic("could not get value from context")
		}

		err := HeaderResponseRewrite(r, c, o)
		if err != nil {
			return o.onError(r, err)
		}

		body, err := BodyResponseRewrite(r, c, o)
		if err != nil {
			return o.onError(r, err)
		}

		for _, m := range o.respMiddlewares {
			if body, err = m(r, body); err != nil {
				return o.onError(r, err)
			}
		}

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
		o.onError = onErr
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
		mutateResPath: func(s string) string {
			return s
		},
		mutateReqPath: func(s string) string {
			return s
		},
	}

	for _, op := range opts {
		op(o)
	}

	return &httputil.ReverseProxy{
		Director:       director(o),
		ModifyResponse: modifyResponse(o),
	}
}