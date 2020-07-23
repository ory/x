package viperx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/pkg/errors"

	"github.com/tidwall/gjson"

	"github.com/ory/jsonschema/v3"
	"github.com/ory/viper"

	"github.com/ory/x/errorsx"
	"github.com/ory/x/jsonschemax"
)

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
		fmt.Printf("Received unexpected error: %+v", err)
	}
}

// PrintHumanReadableValidationErrors prints human readable validation errors. Duh.
func PrintHumanReadableValidationErrors(w io.Writer, err error) {
	var conf bytes.Buffer
	_ = json.NewEncoder(&conf).Encode(viper.AllSettings())
	formatValidationErrorForCLI(w, conf.Bytes(), err)
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

const permOwnerRW = 0o600

func sensitiveDumpAllValues(dir string) error {
	configContent, err := yaml.Marshal(viper.AllSettings())
	if err != nil {
		return errors.WithStack(err)
	}
	return errors.WithStack(
		ioutil.WriteFile(path.Join(dir, fmt.Sprintf("config-%d.yml", time.Now().UnixNano())), configContent, permOwnerRW),
	)
}
