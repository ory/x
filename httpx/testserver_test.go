package httpx

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
)

func TestTestServer(t *testing.T) {
	t.Run("case=fails the test in case of a server failure", func(t *testing.T) {
		s := NewTestServer(&panicFail{}, HandlerFunc(func(t require.TestingT, w http.ResponseWriter, r *http.Request) {
			t.FailNow()
			panic("this panic should not be reached")
		}))
		assert.PanicsWithValue(t, "test failure", func() {
			_, err := s.Client().Get(s.URL)
			require.NoError(t, err)
		})
	})

	t.Run("case=works as a server", func(t *testing.T) {
		s := NewTestServer(t, HandlerFunc(func(_ require.TestingT, w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprintf(w, "OK")
		}))
		for i := 0; i < 10; i++ {
			res, err := s.Client().Get(s.URL)
			require.NoError(t, err)
			body, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			assert.Equal(t, "OK", string(body))
			assert.Equal(t, http.StatusOK, res.StatusCode)
		}
	})

	t.Run("case=supports tls", func(t *testing.T) {
		assert.NotNil(t, NewTestServer(t, nil, WithTLS(true)).TLS)
		assert.Nil(t, NewTestServer(t, nil, WithTLS(false)).TLS)
	})

	t.Run("case=chan handler", func(t *testing.T) {
		h, c := NewTestChanHandler(10)
		for i := 0; i < 10; i++ {
			i := i
			c <- func(_ require.TestingT, w http.ResponseWriter, _ *http.Request) {
				_, _ = fmt.Fprintf(w, "%d", i)
			}
		}
		s := NewTestServer(t, h)
		for i := 0; i < 10; i++ {
			res, err := s.Client().Get(s.URL)
			require.NoError(t, err)
			body, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			assert.Equal(t, fmt.Sprintf("%d", i), string(body))
			assert.Equal(t, http.StatusOK, res.StatusCode)
		}
	})
}

var _ require.TestingT = (*panicFail)(nil)

type panicFail struct{}

func (*panicFail) Errorf(f string, args ...interface{}) {
	fmt.Printf(f, args...)
	fmt.Println()
}

func (*panicFail) FailNow() {
	panic("test failure")
}
