package pagination

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

func header(u *url.URL, rel string, limit, offset int) string {
	q := u.Query()
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	u.RawQuery = q.Encode()
	return fmt.Sprintf("<%s>; rel=\"%s\"", u.String(), rel)
}

// Header returns an http header map, with a single key, "Link". The "Link" header is used in pagination where backwards compatibility is required.
// The Link header will contain links any combination of the first, last, next, or previous (prev) pages in a paginated list (given a limit and an offset, and optionally a total).
// If total is not set, then no "last" page will be calculated.
// If no limit is provided, then this function will return an empty map.
func Header(u *url.URL, total int, limit, offset int) http.Header {
	if limit == 0 {
		return http.Header{}
	}

	// lastOffset will either equal the offset required to contain the remainer,
	// or the limit.
	lastOffset := total + (limit - total%limit) - limit

	// Check for last page
	if offset >= lastOffset {
		if total == 0 {
			return http.Header{
				"Link": []string{
					header(u, "first", limit, 0),
					header(u, "next", limit, ((offset/limit)+1)*limit),
					header(u, "prev", limit, ((offset/limit)-1)*limit),
				},
			}
		}
		return http.Header{
			"Link": []string{
				header(u, "first", limit, 0),
				header(u, "prev", limit, lastOffset-limit),
			},
		}
	}

	if offset < limit {
		return http.Header{
			"Link": []string{
				header(u, "next", limit, limit),
				header(u, "last", limit, lastOffset),
			},
		}
	}

	return http.Header{
		"Link": []string{
			header(u, "first", limit, 0),
			header(u, "next", limit, ((offset/limit)+1)*limit),
			header(u, "prev", limit, ((offset/limit)-1)*limit),
			header(u, "last", limit, lastOffset),
		},
	}

}
