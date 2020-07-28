package watcherx

import (
	"context"
	"github.com/gorilla/websocket"
)

type (
	WebSocketWatcher struct {
		c chan Event
	}
)

func NewWebSocketWatcher(ctx context.Context, url string, c chan Event) (*WebSocketWatcher, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	go forwardWebsocket(ctx, conn, c)
	return &WebSocketWatcher{
		c: c,
	}, nil
}

func forwardWebsocket(ctx context.Context, conn *websocket.Conn, c chan Event) {
	defer conn.Close()
	for {
		_, msg, err := conn.ReadMessage()
		select {
		case <-ctx.Done():
			return
		}
	}
}

func (w *WebSocketWatcher) ID() string {
	panic("implement me")
}

var _ Watcher = &WebSocketWatcher{}
