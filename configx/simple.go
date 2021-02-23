package configx

import (
	"io"
	"io/ioutil"
	"strings"

	"github.com/knadh/koanf"
	"github.com/pkg/errors"

	"github.com/ory/x/stringslice"
)

func StreamToKoanf(i io.Reader, subKey string, parser koanf.Parser) (map[string]interface{}, error) {
	fc, err := ioutil.ReadAll(i)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	v, err := parser.Unmarshal(fc)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if subKey == "" {
		return v, nil
	}

	path := strings.Split(subKey, Delimiter)
	for _, k := range stringslice.Reverse(path) {
		v = map[string]interface{}{
			k: v,
		}
	}

	return v, nil
}
