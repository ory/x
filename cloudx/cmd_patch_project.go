package cloudx

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/ory/x/flagx"

	"github.com/ory/x/cmdx"
)

func NewProjectsPatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project <slug>",
		Args:  cobra.ExactArgs(1),
		Short: "Patch an Ory Cloud Project",
		Example: `ory patch project <your-project-id> \
	--replace '/name="My new project name"' \
	--add '/services/identity/config/courier/smtp={"from_name":"My new email name"}' \
	--replace '/services/identity/config/selfservice/methods/password/enabled=false' \
	--delete '/services/identity/config/selfservice/methods/totp/enabled'`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			h, err := NewSnakeCharmer(cmd)
			if err != nil {
				return err
			}

			files := flagx.MustGetStringSlice(cmd, "file")
			add := flagx.MustGetStringSlice(cmd, "add")
			replace := flagx.MustGetStringSlice(cmd, "replace")
			remove := flagx.MustGetStringSlice(cmd, "remove")

			if len(files)+len(add)+len(replace)+len(remove) == 0 {
				return errors.New("at least one of --file, --add, --replace, or --remove must be set")
			}

			configs, err := ReadConfigFiles(files)
			if err != nil {
				return err
			}

			p, err := h.PatchProject(args[0], configs, add, replace, remove)
			if err != nil {
				return PrintOpenAPIError(cmd, err)
			}

			cmdx.PrintRow(cmd, (*outputProject)(&p.Project))
			for _, warning := range p.Warnings {
				_, _ = fmt.Fprintf(h.verboseErrWriter, "WARNING: %s\n", *warning.Message)
			}

			_, _ = fmt.Fprintln(h.verboseErrWriter, "Project updated successfully!")
			return nil
		},
	}

	cmd.Flags().StringSliceP("file", "f", nil, "Configuration file(s) (file://config.json, https://example.org/config.yaml, ...) to update the project")
	cmd.Flags().StringSlice("replace", nil, "Replace a specific key in the configuration")
	cmd.Flags().StringSlice("add", nil, "Add a specific key to the configuration")
	cmd.Flags().StringSlice("remove", nil, "Remove a specific key from the configuration")
	RegisterYesFlag(cmd.Flags())
	cmdx.RegisterFormatFlags(cmd.Flags())
	return cmd
}
