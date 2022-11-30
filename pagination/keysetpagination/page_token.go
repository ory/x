package keysetpagination

import (
	"fmt"
	"net/url"
	"strings"
)

type PageToken interface {
	Parse(string) map[string]string
	Encode() string
}

var _ PageToken = new(StringPageToken)
var _ PageToken = new(MapPageToken)

type StringPageToken string

func (s StringPageToken) Parse(idField string) map[string]string {
	return map[string]string{idField: string(s)}
}

func (s StringPageToken) Encode() string {
	return string(s)
}

func NewStringPageToken(s string) (PageToken, error) {
	return StringPageToken(s), nil
}

type MapPageToken map[string]string

func (m MapPageToken) Parse(_ string) map[string]string {
	return map[string]string(m)
}

const pageTokenColumnDelim = "/"

func (m MapPageToken) Encode() string {
	elems := []string{}
	for k, v := range m {
		elems = append(elems, fmt.Sprintf("%s=%s", k, v))
	}

	return url.QueryEscape(strings.Join(elems, pageTokenColumnDelim))
}

func NewMapPageToken(s string) (PageToken, error) {
	s, err := url.QueryUnescape(s)
	if err != nil {
		return nil, err
	}
	tokens := strings.Split(s, pageTokenColumnDelim)

	r := map[string]string{}

	for _, p := range tokens {
		if columnName, value, found := strings.Cut(p, "="); found {
			r[columnName] = value
		}
	}

	return MapPageToken(r), nil
}

var _ PageTokenConstructor = NewMapPageToken
var _ PageTokenConstructor = NewStringPageToken

type PageTokenConstructor func(string) (PageToken, error)
