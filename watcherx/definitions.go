package watcherx

import (
	"context"
	"fmt"
	"io"
	"net/url"
)

type (
	Event struct {
		Data  io.Reader
		Src   string
		Error error
	}
	Watcher interface {
		ID() string
	}
	errSchemeUnknown struct {
		scheme string
	}
)

var ErrSchemeUnknown = &errSchemeUnknown{}

func (e *errSchemeUnknown) Is(other error) bool {
	_, ok := other.(*errSchemeUnknown)
	return ok
}

func (e *errSchemeUnknown) Error() string {
	return fmt.Sprintf("unknown scheme '%s' to watch", e.scheme)
}

func CreateWatcher(ctx context.Context, u *url.URL, c chan Event) (Watcher, error) {
	switch u.Scheme {
	case "file":
		return NewFileWatcher(ctx, u.Path, c)
	case "ws":
		return NewWebSocketWatcher(ctx, u.String(), c)
	}
	return nil, &errSchemeUnknown{u.Scheme}
}
