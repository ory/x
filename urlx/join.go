package urlx

import (
	"net/url"

	"github.com/ory/go-convenience/urlx"
	"github.com/ory/x/cmdx"
)

func MustJoin(first string, parts ...string) string {
	u, err := url.Parse(first)
	if err != nil {
		cmdx.Fatalf("Unable to parse %s: %s", first, err)
	}
	return urlx.AppendPaths(u, parts...).String()
}
