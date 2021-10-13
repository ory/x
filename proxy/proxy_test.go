package proxy

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/julienschmidt/httprouter"
	"github.com/ory/herodot"
	"github.com/ory/x/httpx"
	"github.com/ory/x/logrusx"
	"github.com/stretchr/testify/assert"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

type (
	proxyTestCases struct {
		host   string
		method string
		path   string
		status int
	}
	originRequest struct {
		host       string           // the original domain (example.com) requesting to the proxy
		cookieHost string           // the original domain (example.com) cookie domain (can be a sub.sub... domain)
		upstream   *httptest.Server // where the request should end up - (auth1234.another.com)
		shadowHost string           // the proxy host
	}
)

// createUpstreamService creates a testing server
// the server is automatically started and a health endpoint is registered
func createUpstreamService(hw herodot.Writer, routerEndpoints []proxyTestCases) *httptest.Server {
	router := httprouter.New()

	// helper method to register a response writer with a status
	statusWrite := func(status int) func(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
		return func(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
			domain := strings.Split(req.Host, ":")[0]
			http.SetCookie(w, &http.Cookie{
				Name:   "x_secure_session",
				Value:  "1234",
				Path:   "",
				Domain: domain,
			})
			hw.WriteCode(w, req, status, nil)
		}
	}

	// register a health endpoint so that we can check if the service is alive or not
	router.Handle("GET", "/health", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		writer.WriteHeader(http.StatusOK)
	})

	for _, tc := range routerEndpoints {
		router.Handle(tc.method, tc.path, statusWrite(tc.status))
	}

	// create a fake upstream upstreamServer
	upstreamServer := httptest.NewServer(router)
	time.Sleep(time.Second)
	return upstreamServer
}

func duplicateTestCaseToAllMethods(host string, path string, status int) []proxyTestCases {
	methods := []string{"GET", "POST", "PUT", "DELETE"}

	var tc []proxyTestCases

	for _, m := range methods {
		tc = append(tc, proxyTestCases{
			host:   host,
			method: m,
			path:   path,
			status: status,
		})
	}
	return tc
}

func TestRewriteDomain(t *testing.T) {
	// we store our different origin domains here
	// these origin domains will be looked up by the proxy to
	// map it to the correct upstream
	originalRequestDB := map[string]originRequest{
		"auth.example.com": {
			host:       "https://auth.example.com",
			cookieHost: "example.com",
			shadowHost: "https://ory.sh",
		},
		"secure.app.com": {
			host:       "https://secure.app.com",
			cookieHost: "so.secure.app.com",
			shadowHost: "https://ory.sh",
		},
		"example.net": {
			host:       "https://www.example.net",
			cookieHost: "example.net",
			shadowHost: "https://ory.sh",
		},
	}

	var testCases []proxyTestCases

	retryableClient := httpx.NewResilientClient(httpx.ResilientClientWithMaxRetry(10), httpx.ResilientClientWithConnectionTimeout(time.Second))

	for k, or := range originalRequestDB {
		h := herodot.NewJSONWriter(logrusx.New("", ""))

		var originTests []proxyTestCases
		originTests = append(originTests, duplicateTestCaseToAllMethods(or.host, "/random/path", http.StatusOK)...)
		originTests = append(originTests, duplicateTestCaseToAllMethods(or.host, "/an/error", http.StatusInternalServerError)...)

		or.upstream = createUpstreamService(h, originTests)

		req, err := retryablehttp.NewRequest("GET", or.upstream.URL+"/health", nil)
		assert.NoError(t, err)
		_, err = retryableClient.Do(req)
		assert.NoError(t, err)

		t.Logf("Running Upstream Server for (%s) on Address (%s)", or.host, or.upstream.URL)

		originalRequestDB[k] = or

		testCases = append(testCases, originTests...)
	}

	opt := []Options{
		WithLogger(logrusx.New("", "")),
		WithHostMapper(func(host string) (*HostConfig, error) {
			return &HostConfig{
				CookieHost:   originalRequestDB[host].cookieHost,
				UpstreamHost: originalRequestDB[host].upstream.URL,
				OriginalHost: originalRequestDB[host].host,
				ShadowHost:   originalRequestDB[host].shadowHost,
			}, nil
		}),
		WithNegroniMiddleware(func(w http.ResponseWriter, r *http.Request, n http.HandlerFunc) {
			// Disable HSTS because it is very annoying to use in localhost.
			w.Header().Set("Strict-Transport-Security", "max-age=0;")
			n(w, r)
		})}

	// create our proxy service which will forward requests to the upstream server
	proxy := New(opt...)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		addr := proxy.GetServer().Addr
		if addr == "" {
			addr = ":http"
		}
		l, err := net.Listen("tcp", addr)
		assert.NoError(t, err)
		proxy.serverPort = l.Addr().(*net.TCPAddr).Port
		proxy.GetServer().Addr = fmt.Sprintf("127.0.0.1:%d", proxy.serverPort)
		time.Sleep(time.Second)
		wg.Done()
		proxy.GetServer().Serve(l)
	}()

	wg.Wait()

	t.Logf("Running Proxy Server on Address: %s", proxy.GetServer().Addr)

	client := http.DefaultClient

	for _, tc := range testCases {
		// TODO: need to add body requests
		req, err := http.NewRequest(tc.method, "http://"+proxy.GetServer().Addr+tc.path, nil)
		assert.NoError(t, err)
		u, err := url.Parse(tc.host)
		assert.NoError(t, err)
		req.Host = u.Hostname()
		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.EqualValues(t, tc.status, resp.StatusCode, "expected status code %d however received %d", tc.status, resp.StatusCode)
	}

	t.Cleanup(func() {
		proxy.GetServer().Shutdown(context.Background())
		for _, or := range originalRequestDB {
			or.upstream.Close()
		}
	})
}
