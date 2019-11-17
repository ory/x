package httpx

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

func NewRequestJSON(method, url string, data interface{}) (*http.Request, error) {
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(data); err != nil {
		return nil, errors.WithStack(err)
	}
	req, err := http.NewRequest(method, url, &b)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func NewRequestForm(method, url string, data url.Values) (*http.Request, error) {
	req, err := http.NewRequest(method, url, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}

func MustNewRequest(method, url string, body io.Reader, contentType string) *http.Request {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		panic(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req
}
