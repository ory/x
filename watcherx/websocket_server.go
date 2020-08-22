package watcherx

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/gorilla/websocket"

	"github.com/ory/herodot"
)

type eventChannelSlice struct {
	sync.Mutex
	cs []EventChannel
}

var wsClientChannels = eventChannelSlice{}

func WatchAndServeWS(ctx context.Context, u *url.URL, writer herodot.Writer) (http.HandlerFunc, error) {
	c := make(EventChannel)
	if err := Watch(ctx, u, c); err != nil {
		return nil, err
	}
	go broadcaster(ctx, c)
	return serveWS(ctx, writer), nil
}

func broadcaster(ctx context.Context, c EventChannel) {
	defer func() {
		close(c)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case e := <-c:
			wsClientChannels.Lock()
			for _, cc := range wsClientChannels.cs {
				cc <- e
			}
			wsClientChannels.Unlock()
		}
	}
}

func notifyOnClose(ws *websocket.Conn, c chan<- struct{}) {
	for {
		// blocking call to ReadMessage that waits for a close message
		_, _, err := ws.ReadMessage()
		if closeErr, ok := err.(*websocket.CloseError); ok && closeErr.Code == websocket.CloseNormalClosure {
			close(c)
			return
		}
		if opErr, ok := err.(*net.OpError); ok && opErr.Op == "read" && strings.Contains(opErr.Err.Error(), "closed") {
			// the context got canceled and therefore the connection closed
			close(c)
			return
		}
	}
}

func serveWS(ctx context.Context, writer herodot.Writer) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, err := (&websocket.Upgrader{
			ReadBufferSize:  256, // the only message we expect is the close message
			WriteBufferSize: 1024,
		}).Upgrade(w, r, nil)
		if err != nil {
			writer.WriteError(w, r, err)
			return
		}

		// make channel and register it at broadcaster
		c := make(EventChannel)
		wsClientChannels.Lock()
		wsClientChannels.cs = append(wsClientChannels.cs, c)
		wsClientChannels.Unlock()

		wsClosed := make(chan struct{})
		go notifyOnClose(ws, wsClosed)

		defer func() {
			// attempt to close the websocket
			// ignore errors as we are closing everything anyway
			_ = ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "server context canceled"))
			_ = ws.Close()

			wsClientChannels.Lock()
			for i, cc := range wsClientChannels.cs {
				if c == cc {
					wsClientChannels.cs[i] = wsClientChannels.cs[len(wsClientChannels.cs)-1]
					wsClientChannels.cs[len(wsClientChannels.cs)-1] = nil
					wsClientChannels.cs = wsClientChannels.cs[:len(wsClientChannels.cs)-1]
				}
			}
			wsClientChannels.Unlock()
			close(c)
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case <-wsClosed:
				return
			case e, ok := <-c:
				if !ok {
					return
				}
				if err := ws.WriteJSON(e); err != nil {
					return
				}
			}
		}
	}
}
