package proxy

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// This test is a unit test for all the rewrite functions,
// including **all** edge cases. It should not go through the network
// and reverse proxy, but just test all helper functions.

// Things on the TODO:
// - headerResponseRewrite
// - bodyResponseRewrite

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error {
	return nil
}

func TestRewrites(t *testing.T) {
	t.Run("suite=HeaderRequest", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "https://example.com/foo/bar", nil)
		require.NoError(t, err)
		c := &HostConfig{
			CookieDomain:     "example.com",
			originalHost:     "example.com",
			UpstreamHost:     "some-project-1234.oryapis.com",
			UpstreamProtocol: "https",
			PathPrefix:       "/foo",
		}

		headerRequestRewrite(req, c)
		assert.Equal(t, c.UpstreamProtocol, req.URL.Scheme)
		assert.Equal(t, c.UpstreamHost, req.URL.Host)
		assert.Equal(t, "/bar", req.URL.Path)
	})

	t.Run("suite=BodyRequest", func(t *testing.T) {
		t.Run("case=empty body", func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
			require.NoError(t, err)

			newBody, writer, err := bodyRequestRewrite(req, &HostConfig{})
			require.NoError(t, err)
			assert.Nil(t, newBody)
			assert.Nil(t, writer)
		})

		t.Run("case=text replacement without path-prefix", func(t *testing.T) {
			c := &HostConfig{
				originalHost:     "example.com",
				UpstreamProtocol: "https",
				UpstreamHost:     "upstream.ory.sh",
			}

			u := "https://example.com"
			req, err := http.NewRequest(http.MethodPost, u, bytes.NewBufferString(fmt.Sprintf("some text containing the requested URL %s/foo plus a path", u)))
			require.NoError(t, err)

			newBody, _, err := bodyRequestRewrite(req, c)
			assert.Equal(t, fmt.Sprintf("some text containing the requested URL %s://%s/foo plus a path", c.UpstreamProtocol, c.UpstreamHost), string(newBody))
		})

		t.Run("case=text replacement with path-prefix", func(t *testing.T) {
			c := &HostConfig{
				originalHost:     "example.com",
				UpstreamProtocol: "https",
				UpstreamHost:     "upstream.ory.sh",
				PathPrefix:       "/.ory",
			}

			u := "https://example.com"
			req, err := http.NewRequest(http.MethodPost, u, bytes.NewBufferString(fmt.Sprintf("some text containing the requested URL %s/.ory/foo but with a prefix", u)))
			require.NoError(t, err)

			newBody, _, err := bodyRequestRewrite(req, c)
			assert.Equal(t, fmt.Sprintf("some text containing the requested URL %s://%s/foo but with a prefix", c.UpstreamProtocol, c.UpstreamHost), string(newBody))
		})

		t.Run("case=json replacement", func(t *testing.T) {
			c := &HostConfig{
				CookieDomain:     "example.com",
				UpstreamHost:     "upstream.ory.sh",
				PathPrefix:       "/.ory",
				UpstreamProtocol: "https",
				originalHost:     "auth.example.com",
				originalScheme:   "http",
			}

			type bodyDetailsJson struct {
				InnerUrl string `json:"inner_url"`
			}

			type bodyJson struct {
				Url     string          `json:"url"`
				Details bodyDetailsJson `json:"details"`
			}

			body := bodyJson{
				Url: "http://" + c.originalHost + c.PathPrefix,
				Details: bodyDetailsJson{
					InnerUrl: "http://" + c.originalHost + c.PathPrefix + "/path",
				},
			}

			jbody, err := json.Marshal(&body)

			assert.NoError(t, err)

			req, err := http.NewRequest(http.MethodPost, "http://"+c.originalHost, bytes.NewBuffer(jbody))
			require.NoError(t, err)

			newBody, _, err := bodyRequestRewrite(req, c)

			bb := &bodyJson{}
			assert.NoError(t, json.Unmarshal(newBody, &bb))
			assert.Equal(t, "http://"+c.UpstreamHost, bb.Url)
			assert.Equal(t, "http://"+c.UpstreamHost+"/path", bb.Details.InnerUrl)
		})
	})

	t.Run("suit=HeaderResponse", func(t *testing.T) {

		t.Run("case=with location header", func(t *testing.T) {
			upstreamHost := "some-project-1234.oryapis.com"

			c := &HostConfig{
				CookieDomain:     "example.com",
				UpstreamHost:     upstreamHost,
				PathPrefix:       "/foo",
				UpstreamProtocol: "https",
				originalHost:     "example.com",
				originalScheme:   "http",
			}

			header := http.Header{}
			cookie := http.Cookie{
				Name:   "cookie.example",
				Value:  "1234",
				Domain: upstreamHost,
			}

			location := url.URL{
				Scheme: "https",
				Host:   upstreamHost,
				Path:   "/bar",
			}

			header.Set("Set-Cookie", cookie.String())
			header.Set("Location", location.String())

			resp := &http.Response{
				Status:        "ok",
				StatusCode:    200,
				Proto:         "https",
				Header:        header,
				Body:          nil,
				ContentLength: 0,
			}

			err := headerResponseRewrite(resp, c)
			assert.NoError(t, err)

			loc, err := resp.Location()
			assert.NoError(t, err)

			assert.Equal(t, c.originalHost, loc.Host)
			assert.Equal(t, c.originalScheme, loc.Scheme)
			assert.Equal(t, "/foo/bar", loc.Path)

			for _, co := range resp.Cookies() {
				assert.Equal(t, c.CookieDomain, co.Domain)
			}
		})

		t.Run("case=without location header", func(t *testing.T) {
			upstreamHost := "some-project-1234.oryapis.com"

			c := &HostConfig{
				CookieDomain:     "example.com",
				UpstreamHost:     upstreamHost,
				PathPrefix:       "/foo",
				UpstreamProtocol: "https",
				originalHost:     "example.com",
				originalScheme:   "http",
			}

			header := http.Header{}
			cookie := http.Cookie{
				Name:   "cookie.example",
				Value:  "1234",
				Domain: upstreamHost,
			}

			header.Set("Set-Cookie", cookie.String())

			resp := &http.Response{
				Status:        "ok",
				StatusCode:    200,
				Proto:         "https",
				Header:        header,
				Body:          nil,
				ContentLength: 0,
			}

			err := headerResponseRewrite(resp, c)
			assert.NoError(t, err)

			_, err = resp.Location()
			assert.Error(t, err)

			for _, co := range resp.Cookies() {
				assert.Equal(t, c.CookieDomain, co.Domain)
			}
		})

		t.Run("case=without cookie", func(t *testing.T) {
			upstreamHost := "some-project-1234.oryapis.com"

			c := &HostConfig{
				CookieDomain:     "example.com",
				UpstreamHost:     upstreamHost,
				PathPrefix:       "/foo",
				UpstreamProtocol: "https",
				originalHost:     "example.com",
				originalScheme:   "http",
			}

			header := http.Header{}

			resp := &http.Response{
				Status:     "ok",
				StatusCode: 200,
				Proto:      "https",
				Header:     header,
			}

			err := headerResponseRewrite(resp, c)
			assert.NoError(t, err)

			assert.Len(t, resp.Cookies(), 0)
		})

	})

	t.Run("suit=BodyResponse", func(t *testing.T) {

		t.Run("case=empty body", func(t *testing.T) {
			resp := &http.Response{
				Status:        "OK",
				StatusCode:    200,
				Proto:         "http",
				Body:          nil,
				ContentLength: 0,
			}

			_, _, err := bodyResponseRewrite(resp, &HostConfig{})
			assert.NoError(t, err)
		})

		t.Run("case=json body", func(t *testing.T) {
			upstreamHost := "some-project-1234.oryapis.com"

			c := &HostConfig{
				CookieDomain:     "example.com",
				UpstreamHost:     upstreamHost,
				PathPrefix:       "/foo",
				UpstreamProtocol: "http",
				originalHost:     "auth.example.com",
				originalScheme:   "https",
			}

			type bodyRespInner struct {
				InnerKey string `json:"inner_key"`
			}

			type bodyResp struct {
				SomeKey      string          `json:"some_key"`
				InnerRespArr []bodyRespInner `json:"inner_resp_arr"`
				InnerResp    bodyRespInner   `json:"inner_resp"`
			}

			br := bodyResp{
				SomeKey: "https://" + upstreamHost + "/path",
				InnerRespArr: []bodyRespInner{
					{
						InnerKey: "https://" + upstreamHost + "/bar",
					},
				},
				InnerResp: bodyRespInner{
					InnerKey: "https://" + upstreamHost,
				},
			}

			body, err := json.Marshal(&br)
			assert.NoError(t, err)

			resp := &http.Response{
				Status:        "OK",
				StatusCode:    200,
				Proto:         "http",
				Body:          io.NopCloser(bytes.NewReader(body)),
				ContentLength: int64(len(body)),
			}

			b, _, err := bodyResponseRewrite(resp, c)
			assert.NoError(t, err)
			assert.NoError(t, json.Unmarshal(b, &br))
			assert.Equal(t, "https://auth.example.com/foo", br.InnerResp.InnerKey)
			assert.Equal(t, "https://auth.example.com/foo/path", br.SomeKey)
			assert.Equal(t, "https://auth.example.com/foo/bar", br.InnerRespArr[0].InnerKey)
		})

		t.Run("case=string body", func(t *testing.T) {
			upstreamHost := "some-project-1234.oryapis.com"

			c := &HostConfig{
				CookieDomain:     "example.com",
				UpstreamHost:     upstreamHost,
				PathPrefix:       "/foo",
				UpstreamProtocol: "http",
				originalHost:     "auth.example.com",
				originalScheme:   "https",
			}

			bs := fmt.Sprintf("this is a string body https://%s", upstreamHost)

			resp := &http.Response{
				Status:        "OK",
				StatusCode:    200,
				Proto:         "http",
				Body:          io.NopCloser(strings.NewReader(bs)),
				ContentLength: int64(len(bs)),
			}

			b, _, err := bodyResponseRewrite(resp, c)
			assert.NoError(t, err)
			assert.Equal(t, fmt.Sprintf("this is a string body https://%s", c.originalHost+c.PathPrefix), string(b))
		})

	})
}

func TestHelpers(t *testing.T) {
	t.Run("func=stripPort", func(t *testing.T) {
		for input, output := range map[string]string{
			"example.com":      "example.com",
			"example.com:4321": "example.com",
			"192.168.0.0":      "192.168.0.0",
			"192.168.0.0:8080": "192.168.0.0",
		} {
			assert.Equal(t, output, stripPort(input))
		}
	})

	t.Run("func=readBody", func(t *testing.T) {
		t.Run("case=basic body", func(t *testing.T) {
			rawBody, writer, err := readBody(http.Header{}, io.NopCloser(bytes.NewBufferString("simple body")))
			require.NoError(t, err)
			assert.Equal(t, "simple body", string(rawBody))

			_, err = writer.Write([]byte("not compressed"))
			require.NoError(t, err)
			assert.Equal(t, "not compressed", writer.buf.String())
		})

		t.Run("case=gziped body", func(t *testing.T) {
			header := http.Header{}
			header.Set("Content-Encoding", "gzip")
			body := &bytes.Buffer{}
			w := gzip.NewWriter(body)
			_, err := w.Write([]byte("this is compressed"))
			require.NoError(t, err)
			require.NoError(t, w.Close())

			rawBody, writer, err := readBody(header, io.NopCloser(body))
			require.NoError(t, err)
			assert.Equal(t, "this is compressed", string(rawBody))

			_, err = writer.Write([]byte("should compress"))
			assert.NotEqual(t, "should compress", writer.buf.String())

			r, err := gzip.NewReader(&writer.buf)
			require.NoError(t, err)
			content, err := io.ReadAll(r)
			require.NoError(t, err)
			assert.Equal(t, "should compress", string(content))
		})
	})

	t.Run("func=compressableBody.Read", func(t *testing.T) {
		t.Run("case=empty body", func(t *testing.T) {
			n, err := (*compressableBody)(nil).Read(make([]byte, 10))
			assert.True(t, err == io.EOF)
			assert.Equal(t, 0, n)
		})

		t.Run("case=has content", func(t *testing.T) {
			content := "some test content, who cares"
			b := make([]byte, 128)
			n, err := (&compressableBody{
				buf: *bytes.NewBufferString(content),
			}).Read(b)
			require.NoError(t, err)
			assert.Equal(t, content, string(b[:n]))
		})
	})

	t.Run("func=compressableBody.Write", func(t *testing.T) {
		t.Run("case=empty body", func(t *testing.T) {
			n, err := (*compressableBody)(nil).Write([]byte{0, 1, 2, 3})
			assert.NoError(t, err)
			assert.Equal(t, 0, n)
		})

		t.Run("case=no writer", func(t *testing.T) {
			b := &compressableBody{}
			_, err := b.Write([]byte("foo bar"))
			require.NoError(t, err)
			assert.Equal(t, "foo bar", b.buf.String())
		})

		t.Run("case=wrapped writer", func(t *testing.T) {
			other := &bytes.Buffer{}
			b := &compressableBody{}
			b.w = nopWriteCloser{io.MultiWriter(other, &b.buf)}
			_, err := b.Write([]byte("foo bar"))
			require.NoError(t, err)
			assert.Equal(t, "foo bar", b.buf.String())
			assert.Equal(t, "foo bar", other.String())
		})
	})
}
