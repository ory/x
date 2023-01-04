// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package jsonx

import (
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
)

var opAllowList = map[string]struct{}{
	"add":     {},
	"remove":  {},
	"replace": {},
}

func ApplyJSONPatch(p json.RawMessage, object interface{}, denyPaths ...string) error {
	patch, err := jsonpatch.DecodePatch(p)
	if err != nil {
		return err
	}

	denySet := make(map[string]struct{})
	for _, path := range denyPaths {
		denySet[path] = struct{}{}
	}

	for _, op := range patch {
		// Some operations are buggy, see https://github.com/evanphx/json-patch/pull/158
		if _, ok := opAllowList[op.Kind()]; !ok {
			return fmt.Errorf("unsupported operation: %s", op.Kind())
		}

		path, err := op.Path()
		if err != nil {
			return fmt.Errorf("error parsing patch operations: %v", err)
		}
		if _, ok := denySet[path]; ok {
			return fmt.Errorf("patch includes denied path: %s", path)
		}
	}

	original, err := json.Marshal(object)
	if err != nil {
		return err
	}

	modified, err := patch.Apply(original)
	if err != nil {
		return err
	}

	return json.Unmarshal(modified, object)
}
