package watcherx

import (
	"context"
	"fmt"
	"net/url"
)

type (
	errSchemeUnknown struct {
		scheme string
	}
	EventChannel chan Event
	Watcher      interface {
		DispatchNow() error
	}
	dispatcher struct {
		trigger chan struct{}
	}
)

var (
	// ErrSchemeUnknown is just for checking with errors.Is()
	ErrSchemeUnknown     = &errSchemeUnknown{}
	ErrWatcherNotRunning = fmt.Errorf("watcher is not running")
)

func (e *errSchemeUnknown) Is(other error) bool {
	_, ok := other.(*errSchemeUnknown)
	return ok
}

func (e *errSchemeUnknown) Error() string {
	return fmt.Sprintf("unknown scheme '%s' to watch", e.scheme)
}

func newDispatcher() *dispatcher {
	return &dispatcher{
		trigger: make(chan struct{}),
	}
}

func (d *dispatcher) DispatchNow() error {
	if d.trigger == nil {
		return ErrWatcherNotRunning
	}
	d.trigger <- struct{}{}
	return nil
}

func Watch(ctx context.Context, u *url.URL, c EventChannel) (Watcher, error) {
	switch u.Scheme {
	// see urlx.Parse for why the empty string is also file
	case "file", "":
		return WatchFile(ctx, u.Path, c)
	case "ws":
		return WatchWebsocket(ctx, u, c)
	}
	return nil, &errSchemeUnknown{u.Scheme}
}
