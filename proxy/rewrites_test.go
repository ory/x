package proxy

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
)

// This test is a unit test for all the rewrite functions,
// including **all** edge cases. It should not go through the network
// and reverse proxy, but just test all helper functions.

// Things on the TODO:
// - HeaderResponseRewrite
// - BodyResponseRewrite

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

		HeaderRequestRewrite(req, c)
		assert.Equal(t, c.UpstreamProtocol, req.URL.Scheme)
		assert.Equal(t, c.UpstreamHost, req.URL.Host)
		assert.Equal(t, "/bar", req.URL.Path)
	})

	t.Run("suite=BodyRequest", func(t *testing.T) {
		t.Run("case=empty body", func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
			require.NoError(t, err)

			newBody, writer, err := BodyRequestRewrite(req, &HostConfig{})
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

			url := "https://example.com"
			req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(fmt.Sprintf("some text containing the requested URL %s/foo plus a path", url)))
			require.NoError(t, err)

			newBody, _, err := BodyRequestRewrite(req, c)
			assert.Equal(t, fmt.Sprintf("some text containing the requested URL %s://%s/foo plus a path", c.UpstreamProtocol, c.UpstreamHost), string(newBody))
		})

		t.Run("case=text replacement with path-prefix", func(t *testing.T) {
			c := &HostConfig{
				originalHost:     "example.com",
				UpstreamProtocol: "https",
				UpstreamHost:     "upstream.ory.sh",
				PathPrefix:       "/.ory",
			}

			url := "https://example.com"
			req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(fmt.Sprintf("some text containing the requested URL %s/.ory/foo but with a prefix", url)))
			require.NoError(t, err)

			newBody, _, err := BodyRequestRewrite(req, c)
			assert.Equal(t, fmt.Sprintf("some text containing the requested URL %s://%s/foo but with a prefix", c.UpstreamProtocol, c.UpstreamHost), string(newBody))
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
