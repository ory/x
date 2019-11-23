package decoderx

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/santhosh-tekuri/jsonschema/v2"
	"github.com/tidwall/sjson"

	"github.com/ory/herodot"

	"github.com/ory/x/httpx"
	"github.com/ory/x/jsonschemax"
	"github.com/ory/x/stringslice"
)

type (
	// HTTP decodes json and form-data from HTTP Request Bodies.
	HTTP struct{}

	httpDecoderOptions struct {
		allowedContentTypes       []string
		allowedHTTPMethods        []string
		jsonSchemaRef             string
		jsonSchemaCompiler        *jsonschema.Compiler
		jsonSchemaValidate        bool
		maxCircularReferenceDepth uint8
		handleParseErrors         parseErrorStrategy
	}

	// HTTPDecoderOption configures the HTTP decoder.
	HTTPDecoderOption func(*httpDecoderOptions)

	parseErrorStrategy uint8
)

const (
	httpContentTypeMultipartForm  = "multipart/form-data"
	httpContentTypeURLEncodedForm = "application/x-www-form-urlencoded"
	httpContentTypeJSON           = "application/json"
)

const (
	// ParseErrorIgnore will ignore any parse errors caused by strconv.Parse* and use the
	// raw form field value, which is a string, when such a parse error occurs.
	ParseErrorIgnore parseErrorStrategy = iota + 1

	// ParseErrorDefault will ignore any parse errors caused by strconv.Parse* and use the
	// default value of the type to be casted, e.g. float64(0), string("").
	ParseErrorDefault

	// ParseErrorReturn will abort and return with an error if strconv.Parse* returns
	// an error.
	ParseErrorReturn
)

// HTTPFormDecoder configures the HTTP decoder to only accept form-data
// (application/x-www-form-urlencoded, multipart/form-data)
func HTTPFormDecoder() HTTPDecoderOption {
	return func(o *httpDecoderOptions) {
		o.allowedContentTypes = []string{httpContentTypeMultipartForm, httpContentTypeURLEncodedForm}
	}
}

// HTTPJSONDecoder configures the HTTP decoder to only accept form-data
// (application/json).
func HTTPJSONDecoder() HTTPDecoderOption {
	return func(o *httpDecoderOptions) {
		o.allowedContentTypes = []string{httpContentTypeJSON}
	}
}

// HTTPDecoderSetValidatePayloads sets if payloads should be validated or not.
func HTTPDecoderSetValidatePayloads(validate bool) HTTPDecoderOption {
	return func(o *httpDecoderOptions) {
		o.jsonSchemaValidate = validate
	}
}

// HTTPDecoderSetIgnoreParseErrorsStrategy sets a strategy for dealing with strconv.Parse* errors:
//
// - decoderx.ParseErrorIgnore will ignore any parse errors caused by strconv.Parse* and use the
// raw form field value, which is a string, when such a parse error occurs. (default)
// - decoderx.ParseErrorDefault will ignore any parse errors caused by strconv.Parse* and use the
// default value of the type to be casted, e.g. float64(0), string("").
// - decoderx.ParseErrorReturn will abort and return with an error if strconv.Parse* returns
// an error.
func HTTPDecoderSetIgnoreParseErrorsStrategy(strategy parseErrorStrategy) HTTPDecoderOption {
	return func(o *httpDecoderOptions) {
		o.handleParseErrors = strategy
	}
}

// HTTPDecoderSetMaxCircularReferenceDepth sets the maximum recursive reference resolution depth.
func HTTPDecoderSetMaxCircularReferenceDepth(depth uint8) HTTPDecoderOption {
	return func(o *httpDecoderOptions) {
		o.maxCircularReferenceDepth = depth
	}
}

// HTTPJSONSchemaCompiler sets a JSON schema to be used for validation and type assertion of
// incoming requests.
func HTTPJSONSchemaCompiler(ref string, compiler *jsonschema.Compiler) HTTPDecoderOption {
	return func(o *httpDecoderOptions) {
		if compiler == nil {
			compiler = jsonschema.NewCompiler()
		}
		compiler.ExtractAnnotations = true
		o.jsonSchemaCompiler = compiler
		o.jsonSchemaRef = ref
		o.jsonSchemaValidate = true
	}
}

// HTTPRawJSONSchemaCompiler uses a JSON Schema Compiler with the provided JSON Schema in raw byte form.
func HTTPRawJSONSchemaCompiler(raw []byte) (HTTPDecoderOption, error) {
	compiler := jsonschema.NewCompiler()
	id := fmt.Sprintf("%x.json", sha256.Sum256(raw))
	if err := compiler.AddResource(id, bytes.NewReader(raw)); err != nil {
		return nil, err
	}
	compiler.ExtractAnnotations = true

	return func(o *httpDecoderOptions) {
		o.jsonSchemaCompiler = compiler
		o.jsonSchemaRef = id
		o.jsonSchemaValidate = true
	}, nil
}

// MustHTTPRawJSONSchemaCompiler uses HTTPRawJSONSchemaCompiler and panics on error.
func MustHTTPRawJSONSchemaCompiler(raw []byte) HTTPDecoderOption {
	f, err := HTTPRawJSONSchemaCompiler(raw)
	if err != nil {
		panic(err)
	}
	return f
}

func newHTTPDecoderOptions(fs []HTTPDecoderOption) *httpDecoderOptions {
	o := &httpDecoderOptions{
		allowedContentTypes: []string{
			httpContentTypeMultipartForm, httpContentTypeURLEncodedForm, httpContentTypeJSON,
		},
		allowedHTTPMethods:        []string{"POST", "PUT", "PATCH"},
		maxCircularReferenceDepth: 5,
		handleParseErrors:         ParseErrorIgnore,
	}

	for _, f := range fs {
		f(o)
	}

	return o
}

// NewHTTP creates a new HTTP decoder.
func NewHTTP() *HTTP {
	return new(HTTP)
}

func (t *HTTP) validateRequest(r *http.Request, c *httpDecoderOptions) error {
	method := strings.ToUpper(r.Method)

	if !stringslice.Has(c.allowedHTTPMethods, method) {
		return errors.WithStack(herodot.ErrBadRequest.WithReasonf(`Unable to decode body because HTTP Request Method was "%s" but only %v are supported.`, method, c.allowedHTTPMethods))
	}

	if r.ContentLength == 0 {
		return errors.WithStack(herodot.ErrBadRequest.WithReasonf(`Unable to decode HTTP Request Body because its HTTP Header "Content-Length" is zero.`))
	}

	if !httpx.HasContentType(r, c.allowedContentTypes...) {
		return errors.WithStack(herodot.ErrBadRequest.WithReasonf(`HTTP %s Request used unknown HTTP Header "Content-Type: %s", only %v are supported.`, method, r.Header.Get("Content-Type"), c.allowedContentTypes))
	}

	return nil
}

func (t *HTTP) validatePayload(raw json.RawMessage, c *httpDecoderOptions) error {
	if !c.jsonSchemaValidate {
		return nil
	}

	if c.jsonSchemaCompiler == nil {
		return errors.WithStack(herodot.ErrInternalServerError.WithReasonf("JSON Schema Validation is required but no compiler was provided."))
	}

	schema, err := c.jsonSchemaCompiler.Compile(c.jsonSchemaRef)
	if err != nil {
		return errors.WithStack(herodot.ErrInternalServerError.WithReasonf("Unable to load JSON Schema from location: %s", c.jsonSchemaRef).WithDebug(err.Error()))
	}

	if err := schema.Validate(bytes.NewBuffer(raw)); err != nil {
		if _, ok := err.(*jsonschema.ValidationError); ok {
			return errors.WithStack(err)
		}
		return errors.WithStack(herodot.ErrInternalServerError.WithReasonf("Unable to process JSON Schema and input: %s", err).WithDebug(err.Error()))
	}

	return nil
}

// Decode takes a HTTP Request Body and decodes it into destination.
func (t *HTTP) Decode(r *http.Request, destination interface{}, opts ...HTTPDecoderOption) error {
	c := newHTTPDecoderOptions(opts)
	if err := t.validateRequest(r, c); err != nil {
		return err
	}

	if httpx.HasContentType(r, httpContentTypeJSON) {
		return t.decodeJSON(r, destination, c)
	} else if httpx.HasContentType(r, httpContentTypeMultipartForm, httpContentTypeURLEncodedForm) {
		return t.decodeForm(r, destination, c)
	}

	return errors.WithStack(herodot.ErrInternalServerError.WithReasonf("Unable to determine decoder for content type: %s", r.Header.Get("Content-Type")))
}

func (t *HTTP) decodeForm(r *http.Request, destination interface{}, o *httpDecoderOptions) error {
	if o.jsonSchemaCompiler == nil {
		return errors.WithStack(herodot.ErrInternalServerError.WithReasonf("Unable to decode HTTP Form Body because no validation schema was provided. This is a code bug."))
	}

	if err := r.ParseForm(); err != nil {
		return errors.WithStack(herodot.ErrBadRequest.WithReasonf("Unable to decode HTTP %s form body: %s", strings.ToUpper(r.Method), err).WithDebug(err.Error()))
	}

	paths, err := jsonschemax.ListPathsWithRecursion(o.jsonSchemaRef, o.jsonSchemaCompiler, o.maxCircularReferenceDepth)
	if err != nil {
		return errors.WithStack(herodot.ErrInternalServerError.WithTrace(err).WithReasonf("Unable to prepare JSON Schema for HTTP Post Body Form parsing: %s", err).WithDebugf("%+v", err))
	}

	raw := json.RawMessage(`{}`)
	for key := range r.PostForm {
		for _, path := range paths {
			if key == path.Name {
				var err error
				switch path.Type.(type) {
				case []string:
					raw, err = sjson.SetBytes(raw, path.Name, r.PostForm[key])
				case []float64:
					for k, v := range r.PostForm[key] {
						if f, err := strconv.ParseFloat(v, 64); err != nil {
							switch o.handleParseErrors {
							case ParseErrorIgnore:
								raw, err = sjson.SetBytes(raw, path.Name+"."+strconv.Itoa(k), v)
							case ParseErrorDefault:
								raw, err = sjson.SetBytes(raw, path.Name+"."+strconv.Itoa(k), f)
							case ParseErrorReturn:
								return errors.WithStack(herodot.ErrBadRequest.WithReasonf("Expected value to be a number.").
									WithDetail("parse_error", err.Error()).
									WithDetail("name", key).
									WithDetail("index", k).
									WithDetail("value", v))
							}
						} else {
							raw, err = sjson.SetBytes(raw, path.Name+"."+strconv.Itoa(k), f)
						}
					}
				case []bool:
					for k, v := range r.PostForm[key] {
						if f, err := strconv.ParseBool(v); err != nil {
							switch o.handleParseErrors {
							case ParseErrorIgnore:
								raw, err = sjson.SetBytes(raw, path.Name+"."+strconv.Itoa(k), v)
							case ParseErrorDefault:
								raw, err = sjson.SetBytes(raw, path.Name+"."+strconv.Itoa(k), f)
							case ParseErrorReturn:
								return errors.WithStack(herodot.ErrBadRequest.WithReasonf("Expected value to be a boolean.").
									WithDetail("parse_error", err.Error()).
									WithDetail("name", key).
									WithDetail("index", k).
									WithDetail("value", v))
							}
						} else {
							raw, err = sjson.SetBytes(raw, path.Name+"."+strconv.Itoa(k), f)
						}
					}
				case []interface{}:
					raw, err = sjson.SetBytes(raw, path.Name, r.PostForm[key])
				case bool:
					v := r.PostForm[key][len(r.PostForm[key])-1]
					if f, err := strconv.ParseBool(v); err != nil {
						switch o.handleParseErrors {
						case ParseErrorIgnore:
							raw, err = sjson.SetBytes(raw, path.Name, v)
						case ParseErrorDefault:
							raw, err = sjson.SetBytes(raw, path.Name, f)
						case ParseErrorReturn:
							return errors.WithStack(herodot.ErrBadRequest.WithReasonf("Expected value to be a boolean.").
								WithDetail("parse_error", err.Error()).
								WithDetail("name", key).
								WithDetail("value", r.PostForm.Get(key)))
						}
					} else {
						raw, err = sjson.SetBytes(raw, path.Name, f)
					}
				case float64:
					v := r.PostForm.Get(key)
					if f, err := strconv.ParseFloat(v, 64); err != nil {
						switch o.handleParseErrors {
						case ParseErrorIgnore:
							raw, err = sjson.SetBytes(raw, path.Name, v)
						case ParseErrorDefault:
							raw, err = sjson.SetBytes(raw, path.Name, f)
						case ParseErrorReturn:
							return errors.WithStack(herodot.ErrBadRequest.WithReasonf("Expected value to be a number.").
								WithDetail("parse_error", err.Error()).
								WithDetail("name", key).
								WithDetail("value", r.PostForm.Get(key)))
						}
					} else {
						raw, err = sjson.SetBytes(raw, path.Name, f)
					}
				case string:
					raw, err = sjson.SetBytes(raw, path.Name, r.PostForm.Get(key))
				case map[string]interface{}:
					raw, err = sjson.SetBytes(raw, path.Name, r.PostForm.Get(key))
				case []map[string]interface{}:
					raw, err = sjson.SetBytes(raw, path.Name, r.PostForm[key])
				}

				if err != nil {
					return errors.WithStack(herodot.ErrBadRequest.WithReasonf("Unable to type assert values from HTTP Post Body: %s", err))
				}
				break
			}
		}
	}

	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(destination); err != nil {
		return errors.WithStack(herodot.ErrBadRequest.WithReasonf("Unable to decode JSON payload: %s", err))
	}

	if err := t.validatePayload(raw, o); err != nil {
		return err
	}

	return nil
}

func (t *HTTP) decodeJSON(r *http.Request, destination interface{}, o *httpDecoderOptions) error {
	raw, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errors.WithStack(herodot.ErrBadRequest.WithReasonf("Unable to read HTTP POST body: %s", err))
	}

	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(destination); err != nil {
		return errors.WithStack(herodot.ErrBadRequest.WithReasonf("Unable to decode JSON payload: %s", err))
	}

	if err := t.validatePayload(raw, o); err != nil {
		return err
	}

	return nil
}
