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
)

// ErrSchemeUnknown is just for checking with errors.Is()
var ErrSchemeUnknown = &errSchemeUnknown{}

func (e *errSchemeUnknown) Is(other error) bool {
	_, ok := other.(*errSchemeUnknown)
	return ok
}

func (e *errSchemeUnknown) Error() string {
	return fmt.Sprintf("unknown scheme '%s' to watch", e.scheme)
}

func Watch(ctx context.Context, u *url.URL, c EventChannel) error {
	switch u.Scheme {
	case "file":
		return WatchFile(ctx, u.Path, c)
	}
	return &errSchemeUnknown{u.Scheme}
}
