package decoderx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ory/jsonschema/v3"
)

func newRequest(t *testing.T, method, url string, body io.Reader, ct string) *http.Request {
	req, err := http.NewRequest(method, url, body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", ct)
	return req
}

func TestHTTPFormDecoder(t *testing.T) {
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
			request: newRequest(t, "POST", "/", bytes.NewBufferString(`{"foo":"bar", "bar":"baz"}`), httpContentTypeJSON),
			options: []HTTPDecoderOption{HTTPJSONDecoder(), MustHTTPRawJSONSchemaCompiler([]byte(`{
	"$id": "https://example.com/config.schema.json",
	"$schema": "http://json-schema.org/draft-07/schema#",
	"type": "object",
	"properties": {
		"foo": {
			"type": "number"
		},
		"bar": {
			"type": "string"
		}
	}
}`),
			)},
			expectedError: "expected number, but got string",
			expected:      `{ "bar": "baz", "foo": "bar" }`,
		},
		{
			d:       "should pass json with validation",
			request: newRequest(t, "POST", "/", bytes.NewBufferString(`{"foo":"bar"}`), httpContentTypeJSON),
			options: []HTTPDecoderOption{HTTPJSONDecoder(), MustHTTPRawJSONSchemaCompiler([]byte(`{
	"$id": "https://example.com/config.schema.json",
	"$schema": "http://json-schema.org/draft-07/schema#",
	"type": "object",
	"properties": {
		"foo": {
			"type": "string"
		}
	}
}`),
			),
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
			d:             "should fail form request when schema does not validate request",
			request:       newRequest(t, "POST", "/", bytes.NewBufferString(url.Values{"bar": {"bar"}}.Encode()), httpContentTypeURLEncodedForm),
			options:       []HTTPDecoderOption{HTTPJSONSchemaCompiler("stub/schema.json", nil)},
			expectedError: `missing properties: "foo"`,
		},
		{
			d: "should pass form request and type assert data",
			request: newRequest(t, "POST", "/", bytes.NewBufferString(url.Values{
				"name.first": {"Aeneas"},
				"name.last":  {"Rekkas"},
				"age":        {"29"},
				"ratio":      {"0.9"},
				"consent":    {"true"},

				// newsletter represents a special case for checkbox input with true/false and raw HTML.
				"newsletter": {
					"false", // comes from <input type="hidden" name="newsletter" value="false">
					"true",  // comes from <input type="checkbox" name="newsletter" value="true" checked>
				},
			}.Encode()), httpContentTypeURLEncodedForm),
			options: []HTTPDecoderOption{HTTPJSONSchemaCompiler("stub/person.json", nil)},
			expected: `{
	"name": {"first": "Aeneas", "last": "Rekkas"},
	"age": 29,
	"newsletter": true,
	"consent": true,
	"ratio": 0.9
}`,
		},
		{
			d: "should pass JSON request formatted as a form",
			request: newRequest(t, "POST", "/", bytes.NewBufferString(`{
	"name.first": "Aeneas",
	"name.last":  "Rekkas",
	"age":        29,
	"ratio":      0.9,
	"consent":    false,
	"newsletter": true
}`), httpContentTypeJSON),
			options: []HTTPDecoderOption{HTTPDecoderJSONFollowsFormFormat(),
				HTTPJSONSchemaCompiler("stub/person.json", nil)},
			expected: `{
	"name": {"first": "Aeneas", "last": "Rekkas"},
	"age": 29,
	"newsletter": true,
	"consent": false,
	"ratio": 0.9
}`,
		},
		{
			d:       "should fail because json is not an object when using form format",
			request: newRequest(t, "POST", "/", bytes.NewBufferString(`[]`), httpContentTypeJSON),
			options: []HTTPDecoderOption{HTTPDecoderJSONFollowsFormFormat(),
				HTTPJSONSchemaCompiler("stub/person.json", nil)},
			expectedError: "be an object",
		},
		{
			d: "should work with ParseErrorIgnoreConversionErrors",
			request: newRequest(t, "POST", "/", bytes.NewBufferString(url.Values{
				"ratio": {"foobar"},
			}.Encode()), httpContentTypeURLEncodedForm),
			options: []HTTPDecoderOption{
				HTTPJSONSchemaCompiler("stub/person.json", nil),
				HTTPDecoderSetIgnoreParseErrorsStrategy(ParseErrorIgnoreConversionErrors),
				HTTPDecoderSetValidatePayloads(false),
			},
			expected: `{"ratio": "foobar"}`,
		},
		{
			d: "should work with ParseErrorIgnoreConversionErrors",
			request: newRequest(t, "POST", "/", bytes.NewBufferString(url.Values{
				"ratio": {"foobar"},
			}.Encode()), httpContentTypeURLEncodedForm),
			options:  []HTTPDecoderOption{HTTPJSONSchemaCompiler("stub/person.json", nil), HTTPDecoderSetIgnoreParseErrorsStrategy(ParseErrorUseEmptyValueOnConversionErrors)},
			expected: `{"ratio": 0.0}`,
		},
		{
			d: "should work with ParseErrorIgnoreConversionErrors",
			request: newRequest(t, "POST", "/", bytes.NewBufferString(url.Values{
				"ratio": {"foobar"},
			}.Encode()), httpContentTypeURLEncodedForm),
			options:       []HTTPDecoderOption{HTTPJSONSchemaCompiler("stub/person.json", nil), HTTPDecoderSetIgnoreParseErrorsStrategy(ParseErrorReturnOnConversionErrors)},
			expectedError: `strconv.ParseFloat: parsing "foobar"`,
		},
		{
			d: "should interpret numbers as string if mandated by the schema",
			request: newRequest(t, "POST", "/", bytes.NewBufferString(url.Values{
				"name.first": {"12345"},
			}.Encode()), httpContentTypeURLEncodedForm),
			options:  []HTTPDecoderOption{HTTPJSONSchemaCompiler("stub/person.json", nil), HTTPDecoderSetIgnoreParseErrorsStrategy(ParseErrorUseEmptyValueOnConversionErrors)},
			expected: `{"name": {"first": "12345"}}`,
		},
	} {
		t.Run(fmt.Sprintf("case=%d/description=%s", k, tc.d), func(t *testing.T) {
			dec := NewHTTP()
			var destination json.RawMessage
			err := dec.Decode(tc.request, &destination, tc.options...)
			if tc.expectedError != "" {
				if e, ok := errors.Cause(err).(*jsonschema.ValidationError); ok {
					t.Logf("%+v", e)
				}
				require.Error(t, err)
				require.Contains(t, fmt.Sprintf("%+v", err), tc.expectedError)
				if len(tc.expected) > 0 {
					assert.JSONEq(t, tc.expected, string(destination))
				}
				return
			}

			require.NoError(t, err)
			assert.JSONEq(t, tc.expected, string(destination))
		})
	}
}
