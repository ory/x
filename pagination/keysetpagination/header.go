package keysetpagination

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/pkg/errors"
)

// Pagination Request Parameters
//
// The `Link` HTTP header contains multiple links (`first`, `next`, `last`, `previous`) formatted as:
// `<https://{project-slug}.projects.oryapis.com/admin/clients?limit={limit}&offset={offset}>; rel="{page}"`
//
// For details on pagination please head over to the [pagination documentation](https://www.ory.sh/docs/ecosystem/api-design#pagination).
//
// swagger:model tokenPaginationRequestParameters
type RequestParameters struct {
	// Items per Page
	//
	// This is the number of items per page to return.
	// For details on pagination please head over to the [pagination documentation](https://www.ory.sh/docs/ecosystem/api-design#pagination).
	//
	// required: false
	// in: query
	// default: 250
	// min: 1
	// max: 1000
	PageSize int `json:"page_size"`

	// Next Page Token
	//
	// The next page token.
	// For details on pagination please head over to the [pagination documentation](https://www.ory.sh/docs/ecosystem/api-design#pagination).
	//
	// required: false
	// in: query
	PageToken string `json:"page_token"`
}

// Pagination Response Header
//
// The `Link` HTTP header contains multiple links (`first`, `next`, `last`, `previous`) formatted as:
// `<https://{project-slug}.projects.oryapis.com/admin/clients?limit={limit}&offset={offset}>; rel="{page}"`
//
// For details on pagination please head over to the [pagination documentation](https://www.ory.sh/docs/ecosystem/api-design#pagination).
//
// swagger:model tokenPaginationResponseHeaders
type ResponseHeaders struct {
	// The Link HTTP Header
	//
	// The `Link` header contains a comma-delimited list of links to the following pages:
	//
	// - first: The first page of results.
	// - next: The next page of results.
	// - prev: The previous page of results.
	// - last: The last page of results.
	//
	// Pages are omitted if they do not exist. For example, if there is no next page, the `next` link is omitted. Examples:
	//
	//	</clients?limit=5&offset=0>; rel="first",</clients?limit=5&offset=15>; rel="next",</clients?limit=5&offset=5>; rel="prev",</clients?limit=5&offset=20>; rel="last"
	//
	Link string `json:"link"`

	// The X-Total-Count HTTP Header
	//
	// The `X-Total-Count` header contains the total number of items in the collection.
	TotalCount int `json:"x-total-count"`
}

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
