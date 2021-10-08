package proxy

import (
	"context"
	"fmt"
	"github.com/ory/graceful"
	"github.com/ory/herodot"
	"github.com/ory/x/logrusx"
	"github.com/urfave/negroni"
	"net/http"
	"net/http/httputil"
)

type (
	RespMiddleware func(resp *http.Response, body []byte) ([]byte, error)
	ReqMiddleware func(req *http.Request, body []byte) ([]byte, error)
	options struct {
		l               *logrusx.Logger
		hostMapper      func(string) *HostConfig
		onError         func(*http.Response, error) error
		respMiddlewares []RespMiddleware
		reqMiddlewares  []ReqMiddleware
		writer          *herodot.JSONWriter
		serverHost      string
		serverPort      int
		middleware      *negroni.Handler
		server          *http.Server
	}
	Proxy struct {
		*options
	}
	HostConfig struct {
		CookieHost   string
		UpstreamHost string
		OriginalHost string
	}
	Options func(*options)
)

// director is a custom internal function for altering a http.Request
func director(o *options) func(*http.Request) {
	return func(r *http.Request) {
		err := HeaderRequestRewrite(r, o)
		if err != nil {
			o.onError(r.Response, err)
			return
		}
		err = BodyRequestRewrite(r, o)
		if err != nil {
			o.onError(r.Response, err)
			return
		}
	}
}

// modifyResponse is a custom internal function for altering a http.Response
func modifyResponse(o *options) func(*http.Response) error {
	return func(r *http.Response) error {
		err := HeaderResponseRewrite(r, o)
		if err != nil {
			return o.onError(r, err)
		}

		body, err := BodyResponseRewrite(r, o)
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
func New(opts ...Options) *Proxy {
	o := &options{}

	handler := &httputil.ReverseProxy{
		Director:       director(o),
		ModifyResponse: modifyResponse(o),
	}

	for _, op := range opts {
		op(o)
	}

	o.writer = herodot.NewJSONWriter(o.l)

	mw := negroni.New()
	mw.UseHandler(handler)

	o.server = graceful.WithDefaults(&http.Server{
		Addr:    fmt.Sprintf("%s:%d", o.serverHost, o.serverPort),
		Handler: mw,
	})

	return &Proxy{
		o,
	}
}

func (p *Proxy) GetServer() *http.Server {
	return p.server
}

func (p *Proxy) StartServer() {
	if err := graceful.Graceful(func() error {
		return p.server.ListenAndServe()
	}, func(ctx context.Context) error {
		if err := p.server.Shutdown(ctx); err != nil {
			return err
		}
		p.options.l.Println("reverse proxy was shutdown gracefully")
		return nil
	}); err != nil {
		p.options.l.Fatal("failed to gracefully shutdown reverse proxy\n")
	}
}
