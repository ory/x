package cloudx

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"

	"github.com/ory/client-go"
	"github.com/ory/x/cmdx"
)

type bodyer interface {
	Body() []byte
}

func PrintOpenAPIError(cmd *cobra.Command, err error) error {
	if err == nil {
		return nil
	}

	var be bodyer
	if !errors.As(err, &be) {
		return err
	}

	var didPrettyPrint bool
	if message := gjson.GetBytes(be.Body(), "error.message"); message.Exists() {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", message.String())
		didPrettyPrint = true
	}
	if reason := gjson.GetBytes(be.Body(), "error.reason"); reason.Exists() {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", reason.String())
		didPrettyPrint = true
	}

	if didPrettyPrint {
		return cmdx.FailSilently(cmd)
	}

	if body, err := json.MarshalIndent(json.RawMessage(be.Body()), "", "  "); err == nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%s\nFailed to execute API request, see error above.\n", body)
		return cmdx.FailSilently(cmd)
	}

	return err
}

func (h *SnakeCharmer) PrintUpdateProjectWarnings(p *client.SuccessfulProjectUpdate) error {
	if len(p.Warnings) > 0 {
		_, _ = fmt.Fprintln(h.verboseErrWriter)
		_, _ = fmt.Fprintln(h.verboseErrWriter, "Warnings were found.")
		for _, warning := range p.Warnings {
			_, _ = fmt.Fprintf(h.verboseErrWriter, "- %s\n", *warning.Message)
		}
		_, _ = fmt.Fprintln(h.verboseErrWriter, "It is save to ignore these warnings unless your intention was to set these keys.")
	}

	_, _ = fmt.Fprintf(h.verboseErrWriter, "\nProject updated successfully!\n")
	return nil
}
