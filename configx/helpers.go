package configx

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/pflag"
	"github.com/tidwall/gjson"

	"github.com/ory/jsonschema/v3"
	"github.com/ory/x/errorsx"
	"github.com/ory/x/jsonschemax"
)

// RegisterFlags registers the config file flag.
func RegisterFlags(flags *pflag.FlagSet) {
	flags.StringSliceP("config", "c", []string{}, "Path to one or more .json, .yaml, .yml, .toml config files. Values are loaded in the order provided, meaning that the last config file overwrites values from the previous config file.")
}

const permOwnerRW = 0o600

func formatValidationErrorForCLI(w io.Writer, conf []byte, err error) {
	switch e := errorsx.Cause(err).(type) {
	case *jsonschema.ValidationError:
		pointer, validation := jsonschemaFormatError(e)

		if pointer == "#" {
			if len(e.Causes) == 0 {
				_, _ = fmt.Fprintln(w, "(root)")
				_, _ = fmt.Fprintln(w, "^-- "+validation)
				_, _ = fmt.Fprintln(w, "")
			}
		} else {
			spaces := make([]string, len(pointer)+3)
			_, _ = fmt.Fprintf(w, "%s: %+v", pointer, gjson.GetBytes(conf, pointer).Value())
			_, _ = fmt.Fprintln(w, "")
			_, _ = fmt.Fprintf(w, "%s^-- %s", strings.Join(spaces, " "), validation)
			_, _ = fmt.Fprintln(w, "")
			_, _ = fmt.Fprintln(w, "")
		}

		for _, cause := range e.Causes {
			formatValidationErrorForCLI(w, conf, cause)
		}
	default:
		return
	}
}

func jsonschemaFormatError(e *jsonschema.ValidationError) (string, string) {
	var (
		err     error
		pointer string
		message string
	)

	pointer = e.InstancePtr
	message = e.Message
	switch ctx := e.Context.(type) {
	case *jsonschema.ValidationErrorContextRequired:
		if len(ctx.Missing) > 0 {
			message = "one or more required properties are missing"
			pointer = ctx.Missing[0]
		}
	}

	// We can ignore the error as it will simply echo the pointer.
	pointer, err = jsonschemax.JSONPointerToDotNotation(pointer)
	if err != nil {
		pointer = e.InstancePtr
	}

	return pointer, message
}
