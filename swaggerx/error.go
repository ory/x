package swaggerx

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-openapi/runtime"
)

func FormatSwaggerError(err error) string {
	var e *runtime.APIError
	if errors.As(err, &e) {
		body, err := json.Marshal(e.Response)
		if err != nil {
			body = []byte(fmt.Sprintf("%+v", e.Response))
		}

		switch e.Code {
		default:
			return fmt.Sprintf("Unable to complete operation %s because the server responded with status code %d:\n\n%s", e.OperationName, e.Code, body)
		}
	}
	return fmt.Sprintf("%+v", err)
}
