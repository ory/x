package pagination

import (
	"net/http"
	"net/url"
	"testing"
)

func TestHeaders(t *testing.T) {
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

		if expect.Get("Link") != h.Get("Link") {
			t.Fatalf("Unexpected response from Header. Expected %s, got %s", expect.Get("Link"), h.Get("Link"))
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

		if expect.Get("Link") != h.Get("Link") {
			t.Fatalf("Unexpected response from Header. Expected %s, got %s", expect.Get("Link"), h.Get("Link"))
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

		if expect.Get("Link") != h.Get("Link") {
			t.Fatalf("Unexpected response from Header. Expected %s, got %s", expect.Get("Link"), h.Get("Link"))
		}
	})

	t.Run("Create previous, next, first, and last if in the middle", func(t *testing.T) {
		h := Header(u, 300, 50, 150)

		expect := http.Header{
			"Link": []string{
				"<http://example.com?limit=50&offset=100>; rel=\"prev\"",
				"<http://example.com?limit=50&offset=200>; rel=\"next\"",
				"<http://example.com?limit=50&offset=0>; rel=\"first\"",
				"<http://example.com?limit=50&offset=250>; rel=\"last\"",
			},
		}

		if expect.Get("Link") != h.Get("Link") {
			t.Fatalf("Unexpected response from Header. Expected %s, got %s", expect.Get("Link"), h.Get("Link"))
		}
	})
}
