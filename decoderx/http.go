package decoderx

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/tidwall/sjson"

	"github.com/ory/gojsonschema"
	"github.com/ory/herodot"

	"github.com/ory/x/httpx"
	"github.com/ory/x/jsonschemax"
	"github.com/ory/x/stringslice"
)

type (
	HTTP struct{}

	httpDecoderOptions struct {
		allowedContentTypes []string
		allowedHTTPMethods  []string
		jsonSchema          gojsonschema.JSONLoader
	}

	HTTPDecoderOption func(*httpDecoderOptions)
)

const (
	httpContentTypeMultipartForm  = "multipart/form-data"
	httpContentTypeURLEncodedForm = "application/x-www-form-urlencoded"
	httpContentTypeJSON           = "application/json"
)

func HTTPFormDecoder() HTTPDecoderOption {
	return func(o *httpDecoderOptions) {
		o.allowedContentTypes = []string{httpContentTypeMultipartForm, httpContentTypeURLEncodedForm}
	}
}

func HTTPJSONDecoder() HTTPDecoderOption {
	return func(o *httpDecoderOptions) {
		o.allowedContentTypes = []string{httpContentTypeJSON}
	}
}

func HTTPJSONSchema(jl gojsonschema.JSONLoader) HTTPDecoderOption {
	return func(o *httpDecoderOptions) {
		o.jsonSchema = jl
	}
}

func newHTTPDecoderOptions(fs []HTTPDecoderOption) *httpDecoderOptions {
	o := &httpDecoderOptions{
		allowedContentTypes: []string{
			httpContentTypeMultipartForm, httpContentTypeURLEncodedForm, httpContentTypeJSON,
		},
		allowedHTTPMethods: []string{"POST", "PUT", "PATCH"},
	}

	for _, f := range fs {
		f(o)
	}

	return o
}

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
	if c.jsonSchema == nil {
		return nil
	}

	res, err := gojsonschema.Validate(c.jsonSchema, gojsonschema.NewBytesLoader(raw))
	if err != nil {
		return errors.WithStack(herodot.ErrBadRequest.WithReasonf("Unable to load JSON Schema for validation: %s", err).WithDebug(err.Error()))
	}

	if !res.Valid() {
		err := herodot.ErrBadRequest.WithReasonf("HTTP Request Body contains invalid or malformed data")
		for _, e := range res.Errors() {
			err = err.WithDetail(e.Field(), e.Description())
		}

		return errors.WithStack(err)
	}

	return nil
}

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
	if o.jsonSchema != nil {
		return errors.WithStack(herodot.ErrInternalServerError.WithReasonf("Unable to decode HTTP Form Body because no validation schema was provided. This is a code bug."))
	}

	if err := r.ParseForm(); err != nil {
		return errors.WithStack(herodot.ErrBadRequest.WithReasonf("Unable to decode HTTP %s form body: %s", strings.ToUpper(r.Method), err).WithDebug(err.Error()))
	}

	doc, err := o.jsonSchema.LoadJSON()
	if err != nil {
		return errors.WithStack(herodot.ErrInternalServerError.WithReasonf("Unable to load JSON Schema.").WithDebug(err.Error()))
	}

	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(doc); err != nil {
		return errors.WithStack(herodot.ErrInternalServerError.WithReasonf("Unable to encode JSON Schema.").WithDebug(err.Error()))
	}

	paths, err := jsonschemax.ListPathsBytes(b.Bytes())
	if err != nil {
		return errors.WithStack(herodot.ErrInternalServerError.WithTrace(err).WithReasonf("Unable to prepare JSON Schema for HTTP Post Body Form parsing: %s", err).WithDebugf("%+v", err))
	}

	var raw json.RawMessage
	for key := range r.PostForm {
		for _, path := range paths {
			if key == path.Name {
				var err error
				switch path.Type.(type) {
				case []string:
					raw, err = sjson.SetBytes(raw, path.Name, r.PostForm[key])
				case []float64:
					vv := make([]float64, len(r.PostForm[key]))
					for k, v := range r.PostForm[key] {
						f, err := strconv.ParseFloat(v, 64)
						if err != nil {
							return errors.WithStack(herodot.ErrBadRequest.WithReasonf("Expected value to be a number.").
								WithDetail("parse_error", err.Error()).
								WithDetail("name", key).
								WithDetail("index", k).
								WithDetail("value", v))
						}
						vv[k] = f
					}
					raw, err = sjson.SetBytes(raw, path.Name, vv)
				case []bool:
					vv := make([]bool, len(r.PostForm[key]))
					for k, v := range r.PostForm[key] {
						f, err := strconv.ParseBool(v)
						if err != nil {
							return errors.WithStack(herodot.ErrBadRequest.WithReasonf("Expected value to be a boolean.").
								WithDetail("parse_error", err.Error()).
								WithDetail("name", key).
								WithDetail("index", k).
								WithDetail("value", v))
						}
						vv[k] = f
					}
					raw, err = sjson.SetBytes(raw, path.Name, vv)
				case []interface{}:
					raw, err = sjson.SetBytes(raw, path.Name, r.PostForm[key])
				case bool:
					v, err := strconv.ParseBool(r.PostForm.Get(key))
					if err != nil {
						return errors.WithStack(herodot.ErrBadRequest.WithReasonf("Expected value to be a boolean.").
							WithDetail("parse_error", err.Error()).
							WithDetail("name", key).
							WithDetail("value", r.PostForm.Get(key)))
					}
					raw, err = sjson.SetBytes(raw, path.Name, v)
				case float64:
					v, err := strconv.ParseFloat(r.PostForm.Get(key), 64)
					if err != nil {
						return errors.WithStack(herodot.ErrBadRequest.WithReasonf("Expected value to be a number.").
							WithDetail("parse_error", err.Error()).
							WithDetail("name", key).
							WithDetail("value", r.PostForm.Get(key)))
					}
					raw, err = sjson.SetBytes(raw, path.Name, v)
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

	if err := t.validatePayload(raw, o); err != nil {
		return err
	}

	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(destination); err != nil {
		return errors.WithStack(herodot.ErrBadRequest.WithReasonf("Unable to decode JSON payload: %s", err))
	}

	return nil
}

func (t *HTTP) decodeJSON(r *http.Request, destination interface{}, o *httpDecoderOptions) error {
	raw, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errors.WithStack(herodot.ErrBadRequest.WithReasonf("Unable to read HTTP POST body: %s", err))
	}

	if err := t.validatePayload(raw, o); err != nil {
		return err
	}

	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(destination); err != nil {
		return errors.WithStack(herodot.ErrBadRequest.WithReasonf("Unable to decode JSON payload: %s", err))
	}

	return nil
}
