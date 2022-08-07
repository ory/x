package httpx

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
)

// NewTestServer returns a new running test server. It is set up in a way that the client can be used to make requests to the server.
// Test failures in the server will be handled by the client and result in an overall test failure (assuming the client is called only
// in the main test go-routine). It is safe to use `assert` and `require` within the handler function (this is not safe with
// *httptest.Server because a handler is running in a go routine and the *testing.T functions should only be called from the main test routine).
func NewTestServer(t require.TestingT, handler Handler, opts ...ServerOption) *httptest.Server {
	options := &serverOpts{}
	for _, o := range opts {
		o(options)
	}

	newServer := httptest.NewServer
	if options.tlsEnabled {
		newServer = httptest.NewTLSServer
	}

	s := newServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if x := recover(); x != nil && x == failNowCalled {
				// do nothing, the client will call t.FailNow() anyway
			} else if x != nil {
				panic(x)
			}
		}()
		t := &remoteT{t: t, w: w, r: r}
		handler.ServeHTTP(t, t, r)
	}))
	s.Client().Transport = &remoteTestingRoundTripper{
		t:  t,
		rt: s.Client().Transport,
	}

	return s
}

func NewTestChanHandler(buf int) (Handler, chan<- HandlerFunc) {
	c := make(chan HandlerFunc, buf)
	return HandlerFunc(func(t require.TestingT, w http.ResponseWriter, r *http.Request) {
		(<-c)(t, w, r)
	}), c
}

type (
	failNow int
	Handler interface {
		ServeHTTP(t require.TestingT, w http.ResponseWriter, r *http.Request)
	}
)

const (
	failNowCalled     failNow = 1
	statusTestFailure         = 555
)

type HandlerFunc func(t require.TestingT, w http.ResponseWriter, r *http.Request)

func (h HandlerFunc) ServeHTTP(t require.TestingT, w http.ResponseWriter, r *http.Request) {
	h(t, w, r)
}

type remoteT struct {
	w      http.ResponseWriter
	r      *http.Request
	t      require.TestingT
	failed bool
}

var (
	_ assert.TestingT     = (*remoteT)(nil)
	_ require.TestingT    = (*remoteT)(nil)
	_ http.ResponseWriter = (*remoteT)(nil)
)

func (t *remoteT) Errorf(format string, args ...interface{}) {
	t.failed = true
	t.w.WriteHeader(statusTestFailure)
	t.t.Errorf(format, args...)
}

func (t *remoteT) FailNow() {
	t.failed = true
	t.w.WriteHeader(statusTestFailure)
	panic(failNowCalled)
}

func (t *remoteT) Header() http.Header {
	return t.w.Header()
}

func (t *remoteT) Write(i []byte) (int, error) {
	if t.failed {
		return 0, nil
	}
	return t.w.Write(i)
}

func (t *remoteT) WriteHeader(statusCode int) {
	if t.failed {
		return
	}
	t.w.WriteHeader(statusCode)
}

type (
	serverOpts struct {
		tlsEnabled bool
	}
	ServerOption func(*serverOpts)
)

func WithTLS(tlsEnabled bool) ServerOption {
	return func(o *serverOpts) {
		o.tlsEnabled = tlsEnabled
	}
}

type remoteTestingRoundTripper struct {
	t  require.TestingT
	rt http.RoundTripper
}

var _ http.RoundTripper = (*remoteTestingRoundTripper)(nil)

func (rt *remoteTestingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := rt.rt.RoundTrip(req)
	require.NoError(rt.t, err, "%+v %+v", err, resp)

	if resp.StatusCode == statusTestFailure {
		rt.t.Errorf("got test failure from the server, see output above")
		rt.t.FailNow()
	}

	return resp, err
}
