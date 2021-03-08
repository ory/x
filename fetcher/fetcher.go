package fetcher

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"

	"github.com/ory/x/httpx"
)

// Fetcher is able to load file contents from http, https, file, and base64 locations.
type Fetcher struct {
	hc *retryablehttp.Client
}

type opts struct {
	hc *retryablehttp.Client
}

// WithClient sets the http.Client the fetcher uses.
func WithClient(hc *http.Client) func(*opts) {
	return func(o *opts) {
		o.hc = httpx.NewResilientClient(httpx.ResilientClientWithClient(hc))
	}
}

func newOpts() *opts {
	return &opts{
		hc: httpx.NewResilientClient(),
	}
}

// NewFetcher creates a new fetcher instance.
func NewFetcher(opts ...func(*opts)) *Fetcher {
	o := newOpts()
	for _, f := range opts {
		f(o)
	}
	return &Fetcher{hc: o.hc}
}

// Fetch fetches the file contents from the source.
func (f *Fetcher) Fetch(source string) (*bytes.Buffer, error) {
	if strings.HasPrefix(source, "http") || strings.HasPrefix(source, "https") {
		return f.fetchRemote(source)
	} else if strings.HasPrefix(source, "file") {
		return f.fetchFile(strings.Replace(source, "file://", "", 1))
	} else if strings.HasPrefix(source, "base64") {
		src, err := base64.StdEncoding.DecodeString(strings.Replace(source, "base64://", "", 1))
		if err != nil {
			return nil, errors.Wrapf(err, "rule: %s", source)
		}
		return bytes.NewBuffer(src), nil
	}

	return nil, errors.Errorf("source url uses an unknown scheme: %s", source)
}

func (f *Fetcher) fetchRemote(source string) (*bytes.Buffer, error) {
	res, err := f.hc.Get(source)
	if err != nil {
		return nil, errors.Wrapf(err, "rule: %s", source)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf("expected http response status code 200 but got %d when fetching: %s", res.StatusCode, source)
	}

	return f.decode(res.Body)
}

func (f *Fetcher) fetchFile(source string) (*bytes.Buffer, error) {
	fp, err := os.Open(source)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to fetch from source: %s", source)
	}
	defer fp.Close()

	return f.decode(fp)
}

func (f *Fetcher) decode(r io.Reader) (*bytes.Buffer, error) {
	var b bytes.Buffer
	if _, err := io.Copy(&b, r); err != nil {
		return nil, err
	}
	return &b, nil
}
