package keysetpagination

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/pkg/errors"
)

func header(u *url.URL, rel, token string, size int) string {
	q := u.Query()
	q.Set("page_token", token)
	q.Set("page_size", strconv.Itoa(size))
	u.RawQuery = q.Encode()
	return fmt.Sprintf("<%s>; rel=\"%s\"", u.String(), rel)
}

// Header adds the Link header for the page encoded by the paginator.
// It contains links to the first and next page, if one exists.
func Header(w http.ResponseWriter, u *url.URL, p *Paginator) {
	size := p.Size()
	w.Header().Set("Link", header(u, "first", p.defaultToken, size))

	if !p.IsLast() {
		w.Header().Add("Link", header(u, "next", p.Token(), size))
	}
}

// Parse returns the pagination options from the URL query.
func Parse(q *url.Values) ([]Option, error) {
	var opts []Option
	if q.Has("page_token") {
		opts = append(opts, WithToken(q.Get("page_token")))
	}
	if q.Has("page_size") {
		size, err := strconv.Atoi(q.Get("page_size"))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		opts = append(opts, WithSize(size))
	}
	return opts, nil
}
