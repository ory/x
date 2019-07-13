package jsonx

import (
	"encoding/json"
	"io"
)

// NewStrictDecoder is a shorthand for json.Decoder.DisallowUnknownFields
func NewStrictDecoder(b io.Reader) *json.Decoder {
	d := json.NewDecoder(b)
	d.DisallowUnknownFields()
	return d
}
