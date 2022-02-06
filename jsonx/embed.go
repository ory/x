package jsonx

import (
	"encoding/base64"
	"encoding/json"
	"github.com/ory/x/fetcher"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"net/url"
	"strconv"
	"strings"
)

func EmbedSources(in json.RawMessage) (out json.RawMessage, err error) {
	out = make([]byte, len(in))
	copy(out, in)
	if err := embed(gjson.ParseBytes(in), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func embed(parsed gjson.Result, parents []string, result *json.RawMessage) (err error) {
	if parsed.IsObject() {
		parsed.ForEach(func(k, v gjson.Result) bool {
			err = embed(v, append(parents, strings.ReplaceAll(k.String(), ".", "\\.")), result)
			if err != nil {
				return false
			}
			return true
		})
		if err != nil {
			return err
		}
	} else if parsed.IsArray() {
		for kk, vv := range parsed.Array() {
			if err = embed(vv, append(parents, strconv.Itoa(kk)), result); err != nil {
				return err
			}
		}
	} else if parsed.Type != gjson.String {
		return nil
	}

	loc, err := url.ParseRequestURI(parsed.String())
	if err != nil {
		// Not a URL, return
		return nil
	} else if loc.Scheme != "file" && loc.Scheme != "http" && loc.Scheme != "https" && loc.Scheme != "base64" {
		// Not a known pattern, ignore
		return nil
	}

	contents, err := fetcher.NewFetcher().Fetch(loc.String())
	if err != nil {
		return err
	}

	encoded := base64.StdEncoding.EncodeToString(contents.Bytes())
	key := strings.Join(parents, ".")
	if key == "" {
		key = "@"
	}

	interim, err := sjson.SetBytes(*result, key, "base64://"+encoded)
	if err != nil {
		return err
	}

	*result = interim
	return
}
