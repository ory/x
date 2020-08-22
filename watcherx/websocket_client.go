package watcherx

import (
	"context"
	"net/url"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

func WatchWebsocket(ctx context.Context, u *url.URL, c EventChannel) error {
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return errors.WithStack(err)
	}

	wsClosed := make(chan struct{})
	go cleanupOnDone(ctx, conn, c, wsClosed)

	go forwardWebsocketEvents(conn, c, u, wsClosed)

	return nil
}

func cleanupOnDone(ctx context.Context, conn *websocket.Conn, c EventChannel, wsClosed <-chan struct{}) {
	// wait for one of the events to occur
	select {
	case <-ctx.Done():
	case <-wsClosed:
	}

	// clean up channel
	close(c)
	// attempt to close the websocket
	// ignore errors as we are closing everything anyway
	_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "context canceled by server"))
	_ = conn.Close()
}

func forwardWebsocketEvents(ws *websocket.Conn, c EventChannel, u *url.URL, wsClosed chan<- struct{}) {
	serverURL := source(u.String())

	defer func() {
		// this triggers the cleanupOnDone subroutine
		wsClosed <- struct{}{}
	}()

	for {
		// receive messages, this call is blocking
		_, msg, err := ws.ReadMessage()
		if err != nil {
			closeErr, ok := err.(*websocket.CloseError)
			if ok && closeErr.Code == websocket.CloseNormalClosure {
				return
			}
			c <- &ErrorEvent{
				error:  errors.WithStack(err),
				source: serverURL,
			}
			return
		}
		e, err := unmarshalEvent(msg)
		if err != nil {
			c <- &ErrorEvent{
				error:  err,
				source: serverURL,
			}
			continue
		}
		localURL := *u
		localURL.Path = e.Source()
		e.setSource(localURL.String())
		c <- e
	}
}
