package metricsx

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// A Bridge connects Prometheus metrics to a push-based analytics system
type Bridge interface {
	Push(context.Context) error
}

// BridgeManagerOptions contain configuration options for a BridgeManager instance
type BridgeManagerOptions struct {
	Interval time.Duration
	Log      logrus.FieldLogger
}

// The BridgeManager mantains bridge lifecycles for external analytics services
type BridgeManager struct {
	Interval time.Duration
	Log      logrus.FieldLogger
	Bridges  []Bridge
}

// Start begins the process of repeatedly sending metrics to the list of external services.
// It uses a sync.WaitGroup to concurrently push metrics to multiple endpoints, and listens for `context.Done()`
func (b *BridgeManager) Start(ctx context.Context) {
	var (
		t  = time.NewTicker(b.Interval)
		wg = sync.WaitGroup{}
	)

	for {
		select {
		case <-t.C:
			wg.Add(len(b.Bridges))
			for _, v := range b.Bridges {
				go func(r Bridge) {
					defer wg.Done()
					if err := r.Push(ctx); err != nil {
						b.Log.WithError(err).Debug("Unable to send metrics to remote service")
					}
				}(v)
			}
			wg.Wait()

		case <-ctx.Done():
			break
		}
	}
}

// NewBridgeManager returns a BridgeManager and assigns defaults, like a 1 minute interval.
func NewBridgeManager(o *BridgeManagerOptions, b []Bridge) *BridgeManager {
	if o.Interval == time.Duration(0) {
		o.Interval = time.Duration(1 * time.Minute)
	}

	if o.Log == nil {
		o.Log = logrus.New()
	}

	return &BridgeManager{
		Log:      o.Log,
		Interval: o.Interval,
		Bridges:  b,
	}
}
