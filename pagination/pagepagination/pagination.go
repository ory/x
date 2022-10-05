package pagepagination

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type PagePaginator struct {
	MaxItems     int
	DefaultItems int
}

func (p *PagePaginator) defaults() {
	if p.MaxItems == 0 {
		p.MaxItems = 1000
	}

	if p.DefaultItems == 0 {
		p.DefaultItems = 250
	}
}

// swagger:model headerPagePagination
type HeaderAnnotation struct {
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

// ParsePagination parses limit and page from *http.Request with given limits and defaults.
func (p *PagePaginator) ParsePagination(r *http.Request) (page, itemsPerPage int) {
	p.defaults()

	if offsetParam := r.URL.Query().Get("page"); offsetParam == "" {
		page = 0
	} else {
		if offset, err := strconv.ParseInt(offsetParam, 10, 0); err != nil {
			page = 0
		} else {
			page = int(offset)
		}
	}

	if limitParam := r.URL.Query().Get("per_page"); limitParam == "" {
		itemsPerPage = p.DefaultItems
	} else {
		if limit, err := strconv.ParseInt(limitParam, 10, 0); err != nil {
			itemsPerPage = p.DefaultItems
		} else {
			itemsPerPage = int(limit)
		}
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

func header(u *url.URL, rel string, limit, page int64) string {
	q := u.Query()
	q.Set("per_page", fmt.Sprintf("%d", limit))
	q.Set("page", fmt.Sprintf("%d", page/limit))
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
