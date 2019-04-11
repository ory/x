package pagination

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"
)

func TestHeader(t *testing.T) {
	u, err := url.Parse("http://example.com")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("Create previous and first but not next or last if at the end", func(t *testing.T) {
		h := Header(u, 120, 50, 100)

		expect := http.Header{
			"Link": []string{
				"<http://example.com?limit=50&offset=0>; rel=\"first\"",
				"<http://example.com?limit=50&offset=50>; rel=\"prev\"",
			},
		}

		if reflect.DeepEqual(expect, h) != true {
			t.Fatalf("Unexpected response from Header. Expected %+v, got %+v", expect, h)
		}
	})

	t.Run("Create next and last, but not previous or first if at the beginning", func(t *testing.T) {
		h := Header(u, 120, 50, 0)

		expect := http.Header{
			"Link": []string{
				"<http://example.com?limit=50&offset=50>; rel=\"next\"",
				"<http://example.com?limit=50&offset=100>; rel=\"last\"",
			},
		}

		if reflect.DeepEqual(expect, h) != true {
			t.Fatalf("Unexpected response from Header. Expected %+v, got %+v", expect, h)
		}
	})

	t.Run("Create next and last, but not previous or first if on the first page", func(t *testing.T) {
		h := Header(u, 120, 50, 10)

		expect := http.Header{
			"Link": []string{
				"<http://example.com?limit=50&offset=50>; rel=\"next\"",
				"<http://example.com?limit=50&offset=100>; rel=\"last\"",
			},
		}

		if reflect.DeepEqual(expect, h) != true {
			t.Fatalf("Unexpected response from Header. Expected %+v, got %+v", expect, h)
		}
	})

	t.Run("Create previous, next, first, and last if in the middle", func(t *testing.T) {
		h := Header(u, 300, 50, 150)

		expect := http.Header{
			"Link": []string{
				"<http://example.com?limit=50&offset=0>; rel=\"first\"",
				"<http://example.com?limit=50&offset=200>; rel=\"next\"",
				"<http://example.com?limit=50&offset=100>; rel=\"prev\"",
				"<http://example.com?limit=50&offset=250>; rel=\"last\"",
			},
		}

		if expect.Get("Link") != h.Get("Link") {
			t.Fatalf("Unexpected response from Header. Expected %+v, got %+v", expect.Get("Link"), h.Get("Link"))
		}
	})

	t.Run("Header should default limit to 1 no limit was provided", func(t *testing.T) {
		h := Header(u, 100, 0, 20)

		expect := http.Header{
			"Link": []string{
				"<http://example.com?limit=1&offset=0>; rel=\"first\"",
				"<http://example.com?limit=1&offset=21>; rel=\"next\"",
				"<http://example.com?limit=1&offset=19>; rel=\"prev\"",
				"<http://example.com?limit=1&offset=99>; rel=\"last\"",
			},
		}

		if reflect.DeepEqual(expect, h) != true {
			t.Fatalf("Unexpected response from Header. Expected %+v, got %+v", expect, h)
		}
	})

	t.Run("Create previous, next, first, but not last if in the middle and no total was provided", func(t *testing.T) {
		h := Header(u, 0, 50, 150)

		expect := http.Header{
			"Link": []string{
				"<http://example.com?limit=50&offset=0>; rel=\"first\"",
				"<http://example.com?limit=50&offset=200>; rel=\"next\"",
				"<http://example.com?limit=50&offset=100>; rel=\"prev\"",
			},
		}

		if reflect.DeepEqual(expect, h) != true {
			t.Fatalf("Unexpected response from Header. Expected %+v, got %+v", expect, h)
		}
	})
}
