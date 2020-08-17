package watcherx

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"

	"github.com/ory/herodot"
)

func WatchAndServeWS(ctx context.Context, u *url.URL, writer herodot.Writer) (http.HandlerFunc, error) {
	c := make(EventChannel)
	if err := Watch(ctx, u, c); err != nil {
		return nil, err
	}
	return serveWS(ctx, c, writer), nil
}

func serveWS(ctx context.Context, c EventChannel, writer herodot.Writer) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, err := (&websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}).Upgrade(w, r, nil)
		if err != nil {
			writer.WriteError(w, r, err)
			return
		}
		for {
			select {
			case <-ctx.Done():
				return
			case e := <-c:
				if err := ws.WriteJSON(e); err != nil {
					writer.WriteError(w, r, err)
					return
				}
			}
		}
	}
}
