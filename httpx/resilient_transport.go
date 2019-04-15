package httpx

import (
	"context"
	"fmt"
	"net/http"
	"time"
)
import "github.com/cenkalti/backoff"
import "github.com/sirupsen/logrus"
import "github.com/pkg/errors"

var _ http.RoundTripper = new(ResilientRoundTripper)
var errRetry = errors.New("retry")

type retryPolicy func(*http.Response, error) (bool, bool)

// ResilientRoundTripper wraps a RoundTripper and retries requests on failure.
type ResilientRoundTripper struct {
	// RoundTripper the wrapped RoundTripper.
	http.RoundTripper

	// ShouldRetry defines a strategy for retries.
	ShouldRetry retryPolicy

	MaxInterval    time.Duration
	MaxElapsedTime time.Duration
}

func defaultShouldRetry(res *http.Response, err error) (bool, bool) {
	if err != nil || res.StatusCode == 0 || res.StatusCode >= 500 {
		return true, false
	}
	return false, false
}

func LoggedShouldRetry(l logrus.FieldLogger) retryPolicy {
	return func(res *http.Response, err error) (bool, bool) {
		if err != nil {
			l.WithError(err).Errorf("Unable to connect to URL: %s", res.Request.URL.String())
			return true, false
		}
		if res.StatusCode == 0 || res.StatusCode >= 500 {
			l.WithError(errors.New(fmt.Sprintf("received error status code %d", res.StatusCode))).Errorf("Unable to connect to URL: %s", res.Request.URL.String())
			return true, false
		}
		return false, false
	}
}

func NewDefaultResilientRoundTripper(
	maxInterval time.Duration,
	maxElapsedTime time.Duration,
) *ResilientRoundTripper {
	return &ResilientRoundTripper{
		RoundTripper:   http.DefaultTransport,
		ShouldRetry:    defaultShouldRetry,
		MaxInterval:    maxInterval,
		MaxElapsedTime: maxElapsedTime,
	}
}

func NewResilientRoundTripper(
	roundTripper http.RoundTripper,
	maxInterval time.Duration,
	maxElapsedTime time.Duration,
) *ResilientRoundTripper {
	return &ResilientRoundTripper{
		RoundTripper:   roundTripper,
		ShouldRetry:    defaultShouldRetry,
		MaxInterval:    maxInterval,
		MaxElapsedTime: maxElapsedTime,
	}
}

func (rt *ResilientRoundTripper) WithShouldRetry(policy retryPolicy) *ResilientRoundTripper {
	rt.ShouldRetry = policy
	return rt
}

func (rt *ResilientRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithCancel(r.Context())
	bc := backoff.WithContext(&backoff.ExponentialBackOff{
		InitialInterval:     backoff.DefaultInitialInterval,
		RandomizationFactor: backoff.DefaultRandomizationFactor,
		Multiplier:          backoff.DefaultMultiplier,
		Clock:               backoff.SystemClock,
		MaxElapsedTime:      rt.MaxElapsedTime,
		MaxInterval:         rt.MaxInterval,
	}, ctx)
	bc.Reset()

	var res *http.Response
	err := backoff.Retry(func() (err error) {
		res, err = rt.RoundTripper.RoundTrip(r)
		if retry, abort := rt.ShouldRetry(res, err); !abort && retry {
			if err != nil {
				return errors.WithStack(err)
			}
			return errRetry
		}

		cancel()
		return errors.WithStack(err)
	}, bc)

	return res, err
}
