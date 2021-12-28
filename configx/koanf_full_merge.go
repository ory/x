package configx

import (
	"encoding/json"
	"github.com/ory/x/jsonx"
	"github.com/pkg/errors"
	"github.com/tidwall/sjson"
)

func MergeAllTypes(src, dst map[string]interface{}) error {
	rawSrc, err := json.Marshal(src)
	if err != nil {
		return errors.WithStack(err)
	}

	dstSrc, err := json.Marshal(dst)
	if err != nil {
		return errors.WithStack(err)
	}

	keys := jsonx.Flatten(rawSrc)
	for key, value := range keys {
		dstSrc, err = sjson.SetBytes(dstSrc, key, value)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return errors.WithStack(json.Unmarshal(dstSrc, &dst))
}
