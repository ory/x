package tokenpagination

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/tidwall/gjson"

	"github.com/ory/herodot"
)

func encode(offset int64) string {
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"page":"%d","v":1}`, offset)))
}

func decode(s string) (int, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return 0, errors.WithStack(herodot.ErrBadRequest.WithWrap(err).WithReasonf("Unable to parse pagination token: %s", err))
	}

	return int(gjson.Get(string(b), "page").Int()), nil
}

// swagger:model responseHeaderTokenPagination
type ResponseHeaderAnnotation struct {
	// The Link HTTP Header
	//
	// The `Link` header contains a comma-delimited list of links to the following pages:
	//
	// - first: The first page of results.
	// - next: The next page of results.
	// - prev: The previous page of results.
	// - last: The last page of results.
	//
	// Pages are omitted if they do not exist. For example, if there is no next page, the `next` link is omitted.
	//
	//	Example: Link: </clients?limit=5&offset=0>; rel="first",</clients?limit=5&offset=15>; rel="next",</clients?limit=5&offset=5>; rel="prev",</clients?limit=5&offset=20>; rel="last"
	Link string `json:"link"`

	// The X-Total-Count HTTP Header
	//
	// The `X-Total-Count` header contains the total number of items in the collection.
	//
	// Example: 123
	TotalCount int `json:"x-total-count"`
}

type TokenPaginator struct {
	MaxItems     int
	DefaultItems int
}

func (p *TokenPaginator) defaults() {
	if p.MaxItems == 0 {
		p.MaxItems = 1000
	}

	if p.DefaultItems == 0 {
		p.DefaultItems = 250
	}
}

// ParsePagination parses limit and page from *http.Request with given limits and defaults.
func (p *TokenPaginator) ParsePagination(r *http.Request) (page, itemsPerPage int) {
	p.defaults()

	if offsetParam := r.URL.Query().Get("page_token"); len(offsetParam) > 0 {
		page, _ = decode(offsetParam)
	}

	if gotLimit, err := strconv.ParseInt(r.URL.Query().Get("page_size"), 10, 0); err == nil {
		itemsPerPage = int(gotLimit)
	} else {
		itemsPerPage = p.DefaultItems
	}

	if itemsPerPage > p.MaxItems {
		itemsPerPage = p.MaxItems
	}

	if itemsPerPage < 1 {
		itemsPerPage = 1
	}

	if page < 0 {
		page = 0
	}

	return
}

func header(u *url.URL, rel string, itemsPerPage, page int64) string {
	q := u.Query()
	q.Set("page_size", fmt.Sprintf("%d", itemsPerPage))
	q.Set("page_token", encode(page))
	u.RawQuery = q.Encode()
	return fmt.Sprintf("<%s>; rel=\"%s\"", u.String(), rel)
}

func PaginationHeader(w http.ResponseWriter, u *url.URL, total int64, page, itemsPerPage int) {
	if itemsPerPage <= 0 {
		itemsPerPage = 1
	}

	itemsPerPage64 := int64(itemsPerPage)
	offset := int64(page) * itemsPerPage64

	// lastOffset will either equal the offset required to contain the remainder,
	// or the limit.
	var lastOffset int64
	if total%itemsPerPage64 == 0 {
		lastOffset = total - itemsPerPage64
	} else {
		lastOffset = (total / itemsPerPage64) * itemsPerPage64
	}

	w.Header().Set("X-Total-Count", strconv.FormatInt(total, 10))

	// Check for last page
	if offset >= lastOffset {
		if total == 0 {
			w.Header().Set("Link", strings.Join([]string{
				header(u, "first", itemsPerPage64, 0),
				header(u, "next", itemsPerPage64, ((offset/itemsPerPage64)+1)*itemsPerPage64),
				header(u, "prev", itemsPerPage64, ((offset/itemsPerPage64)-1)*itemsPerPage64),
			}, ","))
			return
		}

		if total < itemsPerPage64 {
			w.Header().Set("link", header(u, "first", total, 0))
			return
		}

		w.Header().Set("Link", strings.Join([]string{
			header(u, "first", itemsPerPage64, 0),
			header(u, "prev", itemsPerPage64, lastOffset-itemsPerPage64),
		}, ","))
		return
	}

	if offset < itemsPerPage64 {
		w.Header().Set("Link", strings.Join([]string{
			header(u, "next", itemsPerPage64, itemsPerPage64),
			header(u, "last", itemsPerPage64, lastOffset),
		}, ","))
		return
	}

	w.Header().Set("Link", strings.Join([]string{
		header(u, "first", itemsPerPage64, 0),
		header(u, "next", itemsPerPage64, ((offset/itemsPerPage64)+1)*itemsPerPage64),
		header(u, "prev", itemsPerPage64, ((offset/itemsPerPage64)-1)*itemsPerPage64),
		header(u, "last", itemsPerPage64, lastOffset),
	}, ","))
}
