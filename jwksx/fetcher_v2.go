// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package jwksx

import (
	"context"
	"crypto/sha256"
	"time"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/ory/x/fetcher"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/ory/x/otelx"

	"github.com/dgraph-io/ristretto"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

var ErrUnableToFindKeyID = errors.New("specified JWK kid can not be found in the JWK sets")

type (
	fetcherNextOptions struct {
		forceKID   string
		cacheTTL   time.Duration
		useCache   bool
		httpClient *retryablehttp.Client
	}
	// FetcherNext is a JWK fetcher that can be used to fetch JWKs from multiple locations.
	FetcherNext struct {
		cache *ristretto.Cache
	}
	// FetcherNextOption is a functional option for the FetcherNext.
	FetcherNextOption func(*fetcherNextOptions)
)

// NewFetcherNext returns a new FetcherNext instance.
func NewFetcherNext(cache *ristretto.Cache) *FetcherNext {
	return &FetcherNext{
		cache: cache,
	}
}

// WithForceKID forces the key ID to be used. Required when multiple JWK sets are configured.
func WithForceKID(kid string) FetcherNextOption {
	return func(o *fetcherNextOptions) {
		o.forceKID = kid
	}
}

// WithCacheTTL sets the cache TTL. If not set, the TTL is unlimited.
func WithCacheTTL(ttl time.Duration) FetcherNextOption {
	return func(o *fetcherNextOptions) {
		o.cacheTTL = ttl
	}
}

// WithCacheEnabled enables the cache.
func WithCacheEnabled() FetcherNextOption {
	return func(o *fetcherNextOptions) {
		o.useCache = true
	}
}

// WithHTTPClient will use the given HTTP client to fetch the JSON Web Keys.
func WithHTTPClient(c *retryablehttp.Client) FetcherNextOption {
	return func(o *fetcherNextOptions) {
		o.httpClient = c
	}
}

func (f *FetcherNext) ResolveKey(ctx context.Context, locations string, modifiers ...FetcherNextOption) (jwk.Key, error) {
	return f.ResolveKeyFromLocations(ctx, []string{locations}, modifiers...)
}

func (f *FetcherNext) ResolveKeyFromLocations(ctx context.Context, locations []string, modifiers ...FetcherNextOption) (jwk.Key, error) {
	opts := new(fetcherNextOptions)
	for _, m := range modifiers {
		m(opts)
	}

	if len(locations) > 1 && opts.forceKID == "" {
		return nil, errors.Errorf("a key ID must be specified when multiple JWK sets are configured")
	}

	set := jwk.NewSet()
	eg := new(errgroup.Group)
	for k := range locations {
		location := locations[k]
		eg.Go(func() error {
			remoteSet, err := f.fetch(ctx, location, opts)
			if err != nil {
				return err
			}

			iterator := remoteSet.Iterate(ctx)
			for iterator.Next(ctx) {
				// Pair().Value is always of type jwk.Key when generated by Iterate.
				set.Add(iterator.Pair().Value.(jwk.Key))
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	if opts.forceKID != "" {
		key, found := set.LookupKeyID(opts.forceKID)
		if !found {
			return nil, errors.WithStack(ErrUnableToFindKeyID)
		}

		return key, nil
	}

	// No KID was forced? Use the first key we can find.
	key, found := set.Get(0)
	if !found {
		return nil, errors.WithStack(ErrUnableToFindKeyID)
	}

	return key, nil
}

type JwkParseError struct {
	msg   string
	cause error
}

func (e JwkParseError) Error() string {
	return e.msg
}

func (e JwkParseError) Unwrap() error {
	return e.cause
}

// fetch fetches the JWK set from the given location and if enabled, may use the cache to look up the JWK set.
func (f *FetcherNext) fetch(ctx context.Context, location string, opts *fetcherNextOptions) (_ jwk.Set, err error) {
	tracer := trace.SpanFromContext(ctx).TracerProvider().Tracer("")
	ctx, span := tracer.Start(ctx, "jwksx.FetcherNext.fetch", trace.WithAttributes(attribute.String("location", location)))
	defer otelx.End(span, &err)

	cacheKey := sha256.Sum256([]byte(location))
	if opts.useCache {
		if result, found := f.cache.Get(cacheKey[:]); found {
			return result.(jwk.Set), nil
		}
	}

	var fopts []fetcher.Modifier
	if opts.httpClient != nil {
		fopts = append(fopts, fetcher.WithClient(opts.httpClient))
	}

	result, err := fetcher.NewFetcher(fopts...).FetchContext(ctx, location)
	if err != nil {
		return nil, err
	}

	set, err := jwk.ParseReader(result)
	if err != nil {
		return nil, &JwkParseError{msg: "failed to parse JWK set", cause: err}
	}

	if opts.useCache {
		f.cache.SetWithTTL(cacheKey[:], set, 1, opts.cacheTTL)
	}

	return set, nil
}
