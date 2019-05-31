package jsonx

import (
	"encoding/json"
	"io"
)

func NewStrictDecoder(b io.Reader) *json.Decoder {
	d := json.NewDecoder(b)
	d.DisallowUnknownFields()
	return d
}
