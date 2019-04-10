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

func Header(u *url.URL, count int, limit, offset int) http.Header {
	lastOffset := count + (limit - count%limit) - limit

	// Check for first page
	if offset < limit {
		return http.Header{
			"Link": []string{
				header(u, "next", limit, limit),
				header(u, "last", limit, lastOffset),
			},
		}
	}

	// Check for last page
	if offset >= lastOffset {
		return http.Header{
			"Link": []string{
				header(u, "first", limit, 0),
				header(u, "prev", limit, lastOffset-limit),
			},
		}
	}

	return http.Header{
		"Link": []string{
			header(u, "prev", limit, ((offset/limit)-1)*limit),
			header(u, "next", limit, ((offset/limit)+1)*limit),
			header(u, "first", limit, 0),
			header(u, "last", limit, lastOffset),
		},
	}
}
