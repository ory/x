package watcherx

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/herodot"
	"github.com/ory/x/urlx"
)

func TestWatchWebsocket(t *testing.T) {
	t.Run("case=forwards events", func(t *testing.T) {
		ctx, c, dir, cancel := setup(t)
		defer cancel()
		l, hook := test.NewNullLogger()

		fn := path.Join(dir, "some.file")
		f, err := os.Create(fn)
		require.NoError(t, err)

		handler, err := WatchAndServeWS(ctx, urlx.ParseOrPanic("file://"+fn), herodot.NewJSONWriter(l))
		require.NoError(t, err)
		s := httptest.NewServer(handler)

		u := urlx.ParseOrPanic("ws" + strings.TrimLeft(s.URL, "http"))
		require.NoError(t, WatchWebsocket(ctx, u, c))

		_, err = fmt.Fprint(f, "content here")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		assertChange(t, <-c, "content here", u.String()+fn)

		require.NoError(t, os.Remove(fn))
		assertRemove(t, <-c, u.String()+fn)

		assert.Len(t, hook.Entries, 0, "%+v", hook.Entries)
	})

	t.Run("case=client closes itself on context cancel", func(t *testing.T) {
		ctx1, c, dir, cancel1 := setup(t)
		defer cancel1()
		l, hook := test.NewNullLogger()

		fn := path.Join(dir, "some.file")

		handler, err := WatchAndServeWS(ctx1, urlx.ParseOrPanic("file://"+fn), herodot.NewJSONWriter(l))
		require.NoError(t, err)
		s := httptest.NewServer(handler)

		ctx2, cancel2 := context.WithCancel(context.Background())
		u := urlx.ParseOrPanic("ws" + strings.TrimLeft(s.URL, "http"))
		require.NoError(t, WatchWebsocket(ctx2, u, c))

		cancel2()

		e, ok := <-c
		assert.False(t, ok, "%#v", e)

		assert.Len(t, hook.Entries, 0, "%+v", hook.Entries)
	})

	t.Run("case=quits client watcher when server connection is closed", func(t *testing.T) {
		ctxClient, c, dir, cancel := setup(t)
		defer cancel()
		l, hook := test.NewNullLogger()

		fn := path.Join(dir, "some.file")

		ctxServe, cancelServe := context.WithCancel(context.Background())
		handler, err := WatchAndServeWS(ctxServe, urlx.ParseOrPanic("file://"+fn), herodot.NewJSONWriter(l))
		require.NoError(t, err)
		s := httptest.NewServer(handler)

		u := urlx.ParseOrPanic("ws" + strings.TrimLeft(s.URL, "http"))
		require.NoError(t, WatchWebsocket(ctxClient, u, c))

		cancelServe()

		e, ok := <-c
		assert.False(t, ok, "%#v", e)

		assert.Len(t, hook.Entries, 0, "%+v", hook.Entries)
	})

	t.Run("case=successive watching works after client connection is closed", func(t *testing.T) {
		ctxServer, c, dir, cancel := setup(t)
		defer cancel()
		l, hook := test.NewNullLogger()

		fn := path.Join(dir, "some.file")

		handler, err := WatchAndServeWS(ctxServer, urlx.ParseOrPanic("file://"+fn), herodot.NewJSONWriter(l))
		require.NoError(t, err)
		s := httptest.NewServer(handler)

		ctxClient1, cancelClient1 := context.WithCancel(context.Background())
		u := urlx.ParseOrPanic("ws" + strings.TrimLeft(s.URL, "http"))
		require.NoError(t, WatchWebsocket(ctxClient1, u, c))

		cancelClient1()

		_, ok := <-c
		assert.False(t, ok)

		ctxClient2, cancelClient2 := context.WithCancel(context.Background())
		defer cancelClient2()
		c2 := make(EventChannel)
		require.NoError(t, WatchWebsocket(ctxClient2, u, c2))

		f, err := os.Create(fn)
		require.NoError(t, err)
		require.NoError(t, f.Close())

		assertChange(t, <-c2, "", u.String()+fn)

		assert.Len(t, hook.Entries, 0, "%+v", hook.Entries)
	})

	t.Run("case=broadcasts to multiple client connections", func(t *testing.T) {
		ctxServer, c1, dir, cancel := setup(t)
		defer cancel()
		l, hook := test.NewNullLogger()

		fn := path.Join(dir, "some.file")

		handler, err := WatchAndServeWS(ctxServer, urlx.ParseOrPanic("file://"+fn), herodot.NewJSONWriter(l))
		require.NoError(t, err)
		s := httptest.NewServer(handler)

		ctxClient1, cancelClient1 := context.WithCancel(context.Background())
		defer cancelClient1()

		u := urlx.ParseOrPanic("ws" + strings.TrimLeft(s.URL, "http"))
		require.NoError(t, WatchWebsocket(ctxClient1, u, c1))

		ctxClient2, cancelClient2 := context.WithCancel(context.Background())
		defer cancelClient2()
		c2 := make(EventChannel)
		require.NoError(t, WatchWebsocket(ctxClient2, u, c2))

		f, err := os.Create(fn)
		require.NoError(t, err)
		require.NoError(t, f.Close())

		assertChange(t, <-c1, "", u.String()+fn)
		assertChange(t, <-c2, "", u.String()+fn)

		assert.Len(t, hook.Entries, 0, "%+v", hook.Entries)
	})
}
