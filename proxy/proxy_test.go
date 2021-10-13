package proxy

import (
	"github.com/hashicorp/go-retryablehttp"
	"github.com/julienschmidt/httprouter"
	"github.com/ory/herodot"
	"github.com/ory/x/httpx"
	"github.com/ory/x/logrusx"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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
		"www.example.net": {
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
		WithHostMapper(func(host string) (*HostConfig, error) {
			return &HostConfig{
				CookieHost:   originalRequestDB[host].cookieHost,
				UpstreamHost: originalRequestDB[host].upstream.URL,
				OriginalHost: originalRequestDB[host].host,
				ShadowURL:    originalRequestDB[host].shadowHost,
			}, nil
		})}

	// create our proxy service which will forward requests to the upstream server
	proxy := New(opt...)

	server := httptest.NewServer(proxy)
	client := server.Client()

	t.Logf("Running Proxy Server on Address: %s", server.URL)

	for _, tc := range testCases {
		// TODO: need to add body requests
		req, err := http.NewRequest(tc.method, server.URL+tc.path, nil)
		assert.NoError(t, err)
		u, err := url.Parse(tc.host)
		assert.NoError(t, err)
		req.Host = u.Hostname()
		resp, err := client.Do(req)
		assert.NoError(t, err)
		assert.EqualValues(t, tc.status, resp.StatusCode, "expected status code %d however received %d", tc.status, resp.StatusCode)
	}

	t.Cleanup(func() {
		server.Close()
		for _, or := range originalRequestDB {
			or.upstream.Close()
		}
	})
}

/*func TestBodyRewrite(t *testing.T) {
	var testFiles []struct {
		name string
		path string
		data []byte
	}

	filepath.WalkDir("./stub", func(path string, d fs.DirEntry, err error) error {
		testFiles = append(testFiles, struct {
			name string
			path string
		}{
			d.Name(),
			path,

		})
	}
	return nil)
	rewriteJson()
}*/
