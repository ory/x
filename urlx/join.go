package urlx

import (
	"net/url"

	"github.com/ory/go-convenience/urlx"
	"github.com/ory/x/cmdx"
)

// MustJoin joins the paths of two URLs. Fatals if first is not a URL.
func MustJoin(first string, parts ...string) string {
	u, err := url.Parse(first)
	if err != nil {
		cmdx.Fatalf("Unable to parse %s: %s", first, err)
	}
	return urlx.AppendPaths(u, parts...).String()
}
