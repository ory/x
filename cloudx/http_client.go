package cloudx

import (
	"net/http"
	"time"
)

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

func NewCloudHTTPClient(token string) *http.Client {
	return &http.Client{
		Transport: &tokenTransporter{
			RoundTripper: http.DefaultTransport,
			token:        token,
		},
		Timeout: time.Second * 30,
	}
}
