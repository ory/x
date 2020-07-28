package watcherx

import (
	"context"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

func WatchWebsocket(ctx context.Context, url string, c EventChannel) error {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return errors.WithStack(err)
	}
	go forwardWebsocketEvents(ctx, conn, c)
	return nil
}

func forwardWebsocketEvents(ctx context.Context, conn *websocket.Conn, c EventChannel) {
	defer conn.Close()
	for {

		_, msg, err := conn.ReadMessage()
		select {
		case <-ctx.Done():
			return
		}
	}
}
