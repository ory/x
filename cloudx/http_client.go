package cloudx

import (
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

const projectAccessToken = "ORY_ACCESS_TOKEN"

type tokenTransporter struct {
	http.RoundTripper
	token string
}

func (t *tokenTransporter) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.token != "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	return t.RoundTripper.RoundTrip(req)
}

func NewHTTPClient() *http.Client {
	token := os.Getenv(projectAccessToken)
	c := retryablehttp.NewClient()
	c.Logger = nil

	return &http.Client{
		Transport: &tokenTransporter{
			RoundTripper: c.StandardClient().Transport,
			token:        token,
		},
		Timeout: time.Second * 15,
	}
}
