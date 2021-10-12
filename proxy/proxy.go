package proxy

import (
	"context"
	"fmt"
	"github.com/ory/graceful"
	"github.com/ory/herodot"
	"github.com/ory/x/logrusx"
	"github.com/urfave/negroni"
	"net"
	"net/http"
	"net/http/httputil"
)

type (
	RespMiddleware    func(resp *http.Response, body []byte) ([]byte, error)
	ReqMiddleware     func(req *http.Request, body []byte) ([]byte, error)
	NegroniMiddleware func(w http.ResponseWriter, r *http.Request, n http.HandlerFunc)
	options           struct {
		l                 *logrusx.Logger
		hostMapper        func(string) *HostConfig
		mutateReqPath     func(string) string
		mutateResPath     func(string) string
		onError           func(*http.Response, error) error
		respMiddlewares   []RespMiddleware
		reqMiddlewares    []ReqMiddleware
		negroniMiddleware []NegroniMiddleware
		writer            *herodot.JSONWriter
		serverHost        string
		serverPort        int
		negroni           *negroni.Negroni
		server            *http.Server
	}
	Proxy struct {
		*options
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
		// ShadowHost the host the proxy is imitating
		ShadowHost string
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

func WithNegroniMiddleware(nm ...NegroniMiddleware) Options {
	return func(o *options) {
		o.negroniMiddleware = append(o.negroniMiddleware, nm...)
	}
}

func WithNegroni(n *negroni.Negroni) Options {
	return func(o *options) {
		o.negroni = n
	}
}

// New creates a new Proxy
// A Proxy sets up a middleware with custom request and response modification handlers
func New(opts ...Options) *Proxy {
	o := &options{
		serverHost: "127.0.0.1",
		negroni:    negroni.New(),
	}

	for _, op := range opts {
		op(o)
	}

	handler := &httputil.ReverseProxy{
		Director:       director(o),
		ModifyResponse: modifyResponse(o),
	}

	o.writer = herodot.NewJSONWriter(o.l)

	for _, nm := range o.negroniMiddleware {
		o.negroni.UseFunc(nm)
	}

	o.negroni.UseHandler(handler)

	o.server = graceful.WithDefaults(&http.Server{
		Addr:    fmt.Sprintf("%s:%d", o.serverHost, o.serverPort),
		Handler: o.negroni,
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
		addr := p.server.Addr
		if addr == "" {
			addr = ":http"
		}
		l, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		p.serverPort = l.Addr().(*net.TCPAddr).Port
		p.server.Addr = fmt.Sprintf("%s:%d", p.serverHost, p.serverPort)
		return p.server.Serve(l)
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
