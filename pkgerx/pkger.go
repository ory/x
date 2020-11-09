package pkgerx

import (
	"io/ioutil"

	"github.com/ory/x/ioutilx"

	"github.com/markbates/pkger/pkging"
)

// MustRead reads a pkging.File or panics.
func MustRead(f pkging.File, err error) []byte {
	if err != nil {
		panic(err)
	}
	defer f.Close()
	return ioutilx.MustReadAll(f)
}

// Read reads a pkging.File or returns an error
func Read(f pkging.File, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ioutil.ReadAll(f)
}
