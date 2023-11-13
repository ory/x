// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package fetcher

import (
	"bytes"
	"context"
	"encoding/base64"
	stderrors "errors"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"

	"github.com/ory/x/httpx"
	"github.com/ory/x/stringsx"
)

// Fetcher is able to load file contents from http, https, file, and base64 locations.
type Fetcher struct {
	hc    *retryablehttp.Client
	limit int64
}

type opts struct {
	hc    *retryablehttp.Client
	limit int64
}

var ErrUnknownScheme = stderrors.New("unknown scheme")

// WithClient sets the http.Client the fetcher uses.
func WithClient(hc *retryablehttp.Client) Modifier {
	return func(o *opts) {
		o.hc = hc
	}
}

// WithMaxHTTPMaxBytes reads at most limit bytes from the HTTP response body,
// returning bytes.ErrToLarge if the limit would be exceeded.
func WithMaxHTTPMaxBytes(limit int64) Modifier {
	return func(o *opts) {
		o.limit = limit
	}
}

func newOpts() *opts {
	return &opts{
		hc: httpx.NewResilientClient(),
	}
}

type Modifier func(*opts)

// NewFetcher creates a new fetcher instance.
func NewFetcher(opts ...Modifier) *Fetcher {
	o := newOpts()
	for _, f := range opts {
		f(o)
	}
	return &Fetcher{hc: o.hc, limit: o.limit}
}

// Fetch fetches the file contents from the source.
func (f *Fetcher) Fetch(source string) (*bytes.Buffer, error) {
	return f.FetchContext(context.Background(), source)
}

// FetchContext fetches the file contents from the source and allows to pass a
// context that is used for HTTP requests.
func (f *Fetcher) FetchContext(ctx context.Context, source string) (*bytes.Buffer, error) {
	switch s := stringsx.SwitchPrefix(source); {
	case s.HasPrefix("http://"), s.HasPrefix("https://"):
		return f.fetchRemote(ctx, source)
	case s.HasPrefix("file://"):
		return f.fetchFile(strings.Replace(source, "file://", "", 1))
	case s.HasPrefix("base64://"):
		src, err := base64.StdEncoding.DecodeString(strings.Replace(source, "base64://", "", 1))
		if err != nil {
			return nil, errors.Wrapf(err, "rule: %s", source)
		}
		return bytes.NewBuffer(src), nil
	default:
		return nil, errors.Wrap(ErrUnknownScheme, s.ToUnknownPrefixErr().Error())
	}
}

func (f *Fetcher) fetchRemote(ctx context.Context, source string) (*bytes.Buffer, error) {
	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "rule: %s", source)
	}
	res, err := f.hc.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "rule: %s", source)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf("expected http response status code 200 but got %d when fetching: %s", res.StatusCode, source)
	}

	if f.limit > 0 {
		var buf bytes.Buffer
		n, err := io.Copy(&buf, io.LimitReader(res.Body, f.limit+1))
		if n > f.limit {
			return nil, bytes.ErrTooLarge
		}
		if err != nil {
			return nil, err
		}
		return &buf, nil
	}
	return f.toBuffer(res.Body)
}

func (f *Fetcher) fetchFile(source string) (*bytes.Buffer, error) {
	fp, err := os.Open(source) // #nosec:G304
	if err != nil {
		return nil, errors.Wrapf(err, "unable to fetch from source: %s", source)
	}
	defer func() {
		_ = fp.Close()
	}()

	return f.toBuffer(fp)
}

func (f *Fetcher) toBuffer(r io.Reader) (*bytes.Buffer, error) {
	var b bytes.Buffer
	if _, err := io.Copy(&b, r); err != nil {
		return nil, err
	}
	return &b, nil
}
