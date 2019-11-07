package decoderx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"github.com/ory/gojsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRequest(t *testing.T, method, url string, body io.Reader, ct string) *http.Request {
	req, err := http.NewRequest(method, url, body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", ct)
	return req
}

func readFile(t *testing.T, path string) string {
	f, err := ioutil.ReadFile(path)
	require.NoError(t, err)
	return string(f)
}

func TestHTTPFormDecoder(t *testing.T) {
	dec := NewHTTP()
	for k, tc := range []struct {
		d             string
		request       *http.Request
		contentType   string
		options       []HTTPDecoderOption
		expected      string
		expectedError string
	}{
		{
			d:             "should fail because the method is GET",
			request:       &http.Request{Header: map[string][]string{}, Method: "GET"},
			expectedError: "HTTP Request Method",
		},
		{
			d:             "should fail because the body is empty",
			request:       &http.Request{Header: map[string][]string{}, Method: "POST"},
			expectedError: "Content-Length",
		},
		{
			d:             "should fail because content type is missing",
			request:       newRequest(t, "POST", "/", nil, ""),
			expectedError: "Content-Length",
		},
		{
			d:             "should fail because content type is missing",
			request:       newRequest(t, "POST", "/", bytes.NewBufferString("foo"), ""),
			expectedError: "Content-Type",
		},
		{
			d:        "should pass with json without validation",
			request:  newRequest(t, "POST", "/", bytes.NewBufferString(`{"foo":"bar"}`), httpContentTypeJSON),
			expected: `{"foo":"bar"}`,
		},
		{
			d:             "should fail json if content type is not accepted",
			request:       newRequest(t, "POST", "/", bytes.NewBufferString(`{"foo":"bar"}`), httpContentTypeJSON),
			options:       []HTTPDecoderOption{HTTPFormDecoder()},
			expectedError: "Content-Type: application/json",
		},
		{
			d:       "should fail json if validation fails",
			request: newRequest(t, "POST", "/", bytes.NewBufferString(`{"foo":"bar"}`), httpContentTypeJSON),
			options: []HTTPDecoderOption{HTTPJSONDecoder(), HTTPJSONSchema(
				gojsonschema.NewStringLoader(`{
	"$id": "https://example.com/config.schema.json",
	"$schema": "http://json-schema.org/draft-07/schema#",
	"type": "object",
	"properties": {
		"foo": {
			"type": "number"
		}
	}
}`),
			)},
			expectedError: "Invalid type",
		},
		{
			d:       "should pass json with validation",
			request: newRequest(t, "POST", "/", bytes.NewBufferString(`{"foo":"bar"}`), httpContentTypeJSON),
			options: []HTTPDecoderOption{HTTPJSONDecoder(), HTTPJSONSchema(
				gojsonschema.NewStringLoader(`{
	"$id": "https://example.com/config.schema.json",
	"$schema": "http://json-schema.org/draft-07/schema#",
	"type": "object",
	"properties": {
		"foo": {
			"type": "string"
		}
	}
}`,
				)),
			},
			expected: `{"foo":"bar"}`,
		},
		{
			d:             "should fail form request when form is used but only json is allowed",
			request:       newRequest(t, "POST", "/", bytes.NewBufferString(url.Values{"foo": {"bar"}}.Encode()), httpContentTypeURLEncodedForm),
			options:       []HTTPDecoderOption{HTTPJSONDecoder()},
			expectedError: "Content-Type: application/x-www-form-urlencoded",
		},
		{
			d:             "should fail form request when schema is missing",
			request:       newRequest(t, "POST", "/", bytes.NewBufferString(url.Values{"foo": {"bar"}}.Encode()), httpContentTypeURLEncodedForm),
			options:       []HTTPDecoderOption{},
			expectedError: "no validation schema was provided",
		},
		{
			d:       "should fail form request when schema does not validate request",
			request: newRequest(t, "POST", "/", bytes.NewBufferString(url.Values{"foo": {"bar"}}.Encode()), httpContentTypeURLEncodedForm),
			options: []HTTPDecoderOption{HTTPJSONSchema(
				gojsonschema.NewReferenceLoader("file://./stub/schema.json")),
			},
			expected: `{
	"foo": "bar"
}`,
		},
	} {
		t.Run(fmt.Sprintf("case=%d/description=%s", k, tc.d), func(t *testing.T) {
			var destination json.RawMessage
			err := dec.Decode(tc.request, &destination, tc.options...)
			if tc.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, fmt.Sprintf("%+v", err), tc.expectedError)
				return
			}

			require.NoError(t, err)
			assert.JSONEq(t, tc.expected, string(destination))
		})
	}
}
