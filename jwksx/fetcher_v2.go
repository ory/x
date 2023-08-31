package jwksx

import (
	"context"
	"crypto/sha256"
	"github.com/dgraph-io/ristretto"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/ory/x/fetcher"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"net/url"
	"time"
)

var ErrUnableToFindKeyID = errors.New("specified JWK kid can not be found in the JWK sets")

type (
	fetcherNextOptions struct {
		forceKID string
		cacheTTL time.Duration
		useCache bool
	}
	// FetcherNext is a JWK fetcher that can be used to fetch JWKs from multiple locations.
	FetcherNext struct {
		// Fetcher is the underlying https/file/base64 fetcher used to fetch the byte stream.
		Fetcher *fetcher.Fetcher `json:"fetcher,omitempty"`
		cache   *ristretto.Cache
	}
	// FetcherNextOption is a functional option for the FetcherNext.
	FetcherNextOption func(*fetcherNextOptions)
)

// NewFetcherNext returns a new FetcherNext instance.
func NewFetcherNext(fetcher *fetcher.Fetcher, cache *ristretto.Cache) *FetcherNext {
	return &FetcherNext{
		Fetcher: fetcher,
		cache:   cache,
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

func (f *FetcherNext) ResolveKey(ctx context.Context, locations []url.URL, modifiers ...FetcherNextOption) (jwk.Key, error) {
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
			remoteSet, err := f.fetch(ctx, location.String(), opts)
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

// fetch fetches the JWK set from the given location and if enabled, may use the cache to look up the JWK set.
func (f *FetcherNext) fetch(ctx context.Context, location string, opts *fetcherNextOptions) (jwk.Set, error) {
	cacheKey := sha256.Sum256([]byte(location))
	if opts.useCache {
		if result, found := f.cache.Get(cacheKey); found {
			return result.(jwk.Set), nil
		}
	}

	result, err := f.Fetcher.FetchContext(ctx, location)
	if err != nil {
		return nil, err
	}

	set, err := jwk.ParseReader(result)
	if err != nil {
		return nil, err
	}

	if opts.useCache {
		f.cache.SetWithTTL(sha256.Sum256([]byte(location)), set, 1, opts.cacheTTL)
	}

	return set, nil
}
