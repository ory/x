package ioutilx

import (
	"io"
	"io/ioutil"
)

// MustReadAll reads a reader or panics.
func MustReadAll(r io.Reader) []byte {
	all, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}
	return all
}
