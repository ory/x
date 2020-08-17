package watcherx

import (
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
}
