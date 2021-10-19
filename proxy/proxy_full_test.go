package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/x/urlx"
)

// This test is a full integration test for the proxy.
// It does not have to cover **all** edge cases included in the rewrite
// unit test, but should use all features like path prefix, ...

// Things on the TODO:
// - Test onError function

const statusTestFailure = 555

type (
	handler     func(w http.ResponseWriter, r *http.Request)
	chanHandler struct {
		handlers chan handler
	}
	remoteT struct {
		w      http.ResponseWriter
		r      *http.Request
		t      *testing.T
		failed bool
	}
	testingRoundTripper struct {
		t  *testing.T
		rt http.RoundTripper
	}
)

func (h *chanHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/health/ready" {
		w.WriteHeader(http.StatusOK)
		return
	}
	(<-h.handlers)(w, r)
}

func (t *remoteT) Errorf(format string, args ...interface{}) {
	t.failed = true
	t.w.WriteHeader(statusTestFailure)
	t.t.Errorf(format, args...)
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

func (rt *testingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := rt.rt.RoundTrip(req)
	require.NoError(rt.t, err)

	if resp.StatusCode == statusTestFailure {
		rt.t.Error("got test failure from the server, see output above")
		rt.t.FailNow()
	}

	return resp, err
}

func TestFullIntegration(t *testing.T) {
	upstreamHandler := &chanHandler{
		handlers: make(chan handler),
	}
	upstreamServer := httptest.NewTLSServer(upstreamHandler)
	defer upstreamServer.Close()

	// create the proxy
	hostMapper := make(chan func(host string) (*HostConfig, error))
	reqMiddleware := make(chan ReqMiddleware)
	respMiddleware := make(chan RespMiddleware)

	proxy := httptest.NewTLSServer(New(
		func(_ context.Context, host string) (*HostConfig, error) {
			return (<-hostMapper)(host)
		},
		WithTransport(upstreamServer.Client().Transport),
		WithReqMiddleware(func(req *http.Request, body []byte) ([]byte, error) {
			f := <-reqMiddleware
			if f == nil {
				return body, nil
			}
			return f(req, body)
		}),
		WithRespMiddleware(func(resp *http.Response, body []byte) ([]byte, error) {
			f := <-respMiddleware
			if f == nil {
				return body, nil
			}
			return f(resp, body)
		})))
	cl := proxy.Client()
	cl.Transport = &testingRoundTripper{t, cl.Transport}
	cl.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	for _, tc := range []struct {
		desc           string
		hostMapper     func(host string) (*HostConfig, error)
		handler        func(assert *assert.Assertions, w http.ResponseWriter, r *http.Request)
		request        func(t *testing.T) *http.Request
		assertResponse func(t *testing.T, r *http.Response)
		reqMiddleware  ReqMiddleware
		respMiddleware RespMiddleware
	}{
		{
			desc: "body replacement",
			hostMapper: func(host string) (*HostConfig, error) {
				if host != "example.com" {
					return nil, fmt.Errorf("got unexpected host %s, expected 'example.com'", host)
				}
				return &HostConfig{
					CookieDomain: "example.com",
					PathPrefix:   "/foo",
				}, nil
			},
			handler: func(assert *assert.Assertions, w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				assert.NoError(err)
				assert.Equal(fmt.Sprintf("some random content containing the request URL and path prefix %s/bar but also other stuff", upstreamServer.URL), string(body))

				_, err = w.Write([]byte(fmt.Sprintf("just responding with my own URL: %s/baz and some path of course", upstreamServer.URL)))
				assert.NoError(err)
			},
			request: func(t *testing.T) *http.Request {
				req, err := http.NewRequest(http.MethodPost, proxy.URL+"/foo", bytes.NewBufferString("some random content containing the request URL and path prefix https://example.com/foo/bar but also other stuff"))
				require.NoError(t, err)
				req.Host = "example.com"
				return req
			},
			assertResponse: func(t *testing.T, resp *http.Response) {
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Equal(t, "just responding with my own URL: https://example.com/foo/baz and some path of course", string(body))
			},
		},
		{
			desc: "redirection replacement",
			hostMapper: func(host string) (*HostConfig, error) {
				if host != "redirect.me" {
					return nil, fmt.Errorf("got unexpected host %s, expected 'redirect.me'", host)
				}
				return &HostConfig{
					CookieDomain: "redirect.me",
				}, nil
			},
			handler: func(_ *assert.Assertions, w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, upstreamServer.URL+"/redirection/target", http.StatusSeeOther)
			},
			request: func(t *testing.T) *http.Request {
				req, err := http.NewRequest(http.MethodGet, proxy.URL, nil)
				require.NoError(t, err)
				req.Host = "redirect.me"
				return req
			},
			assertResponse: func(t *testing.T, r *http.Response) {
				assert.Equal(t, http.StatusSeeOther, r.StatusCode)
				assert.Equal(t, "https://redirect.me/redirection/target", r.Header.Get("Location"))
			},
		},
		{
			desc: "cookie replacement",
			hostMapper: func(host string) (*HostConfig, error) {
				if host != "auth.cookie.love" {
					return nil, fmt.Errorf("got unexpected host %s, expected 'cookie.love'", host)
				}
				return &HostConfig{
					CookieDomain: "cookie.love",
				}, nil
			},
			handler: func(assert *assert.Assertions, w http.ResponseWriter, r *http.Request) {
				http.SetCookie(w, &http.Cookie{
					Name:   "auth",
					Value:  "my random cookie",
					Domain: urlx.ParseOrPanic(upstreamServer.URL).Hostname(),
				})
				_, err := w.Write([]byte("OK"))
				assert.NoError(err)
			},
			request: func(t *testing.T) *http.Request {
				req, err := http.NewRequest(http.MethodGet, proxy.URL, nil)
				require.NoError(t, err)
				req.Host = "auth.cookie.love"
				return req
			},
			assertResponse: func(t *testing.T, r *http.Response) {
				cookies := r.Cookies()
				require.Len(t, cookies, 1)
				c := cookies[0]
				assert.Equal(t, "auth", c.Name)
				assert.Equal(t, "my random cookie", c.Value)
				assert.Equal(t, "cookie.love", c.Domain)
			},
		},
		{
			desc: "custom middleware",
			hostMapper: func(host string) (*HostConfig, error) {
				return &HostConfig{}, nil
			},
			handler: func(assert *assert.Assertions, w http.ResponseWriter, r *http.Request) {
				assert.Equal("noauth.example.com", r.Host)
				b, err := ioutil.ReadAll(r.Body)
				assert.NoError(err)
				assert.Equal("this is a new body", string(b))

				_, err = w.Write([]byte("OK"))
				assert.NoError(err)
			},
			request: func(t *testing.T) *http.Request {
				req, err := http.NewRequest(http.MethodPost, proxy.URL, bytes.NewReader([]byte("body")))
				require.NoError(t, err)
				req.Host = "auth.example.com"
				return req
			},
			assertResponse: func(t *testing.T, r *http.Response) {
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				assert.Equal(t, "OK", string(body))
				assert.Equal(t, "1234", r.Header.Get("Some-Header"))
			},
			reqMiddleware: func(req *http.Request, body []byte) ([]byte, error) {
				req.Host = "noauth.example.com"
				body = []byte("this is a new body")
				return body, nil
			},
			respMiddleware: func(resp *http.Response, body []byte) ([]byte, error) {
				resp.Header.Add("Some-Header", "1234")
				return body, nil
			},
		},
	} {
		t.Run("case="+tc.desc, func(t *testing.T) {
			go func() {
				hostMapper <- func(host string) (*HostConfig, error) {
					hc, err := tc.hostMapper(host)
					if err == nil {
						hc.UpstreamHost = urlx.ParseOrPanic(upstreamServer.URL).Host
						hc.UpstreamProtocol = urlx.ParseOrPanic(upstreamServer.URL).Scheme
					}
					return hc, err
				}
				reqMiddleware <- tc.reqMiddleware

				upstreamHandler.handlers <- func(w http.ResponseWriter, r *http.Request) {
					t := &remoteT{t: t, w: w, r: r}
					tc.handler(assert.New(t), t, r)
				}

				respMiddleware <- tc.respMiddleware
			}()
			resp, err := cl.Do(tc.request(t))
			require.NoError(t, err)
			tc.assertResponse(t, resp)
		})
	}
}
