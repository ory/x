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
	go closeOnDone(ctx, conn)
	go forwardWebsocketEvents(conn, c, u)
	return nil
}

func closeOnDone(ctx context.Context, conn *websocket.Conn) {
	<-ctx.Done()
	conn.Close()
}

func forwardWebsocketEvents(conn *websocket.Conn, c EventChannel, u *url.URL) {
	serverURL := source(u.String())
	defer conn.Close()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			c <- &ErrorEvent{
				error:  errors.WithStack(err),
				source: serverURL,
			}
			continue
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
