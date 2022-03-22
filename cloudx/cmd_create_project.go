package cloudx

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/ory/x/cmdx"
	"github.com/ory/x/flagx"
)

func NewCreateProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: fmt.Sprintf("Create a new Ory Cloud Project"),
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := NewSnakeCharmer(cmd)
			if err != nil {
				return err
			}

			name := flagx.MustGetString(cmd, "name")
			if len(name) == 0 && flagx.MustGetBool(cmd, string(cmdx.FormatQuiet)) {
				return errors.New("you must specify the --name flag when using --quiet")
			}

			stdin := h.Stdin()
			for name == "" {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Enter a name for your project: ")
				name, err = stdin.ReadString('\n')
				if err != nil {
					return errors.Wrap(err, "failed to read from stdin")
				}
			}

			p, err := h.CreateProject(name)
			if err != nil {
				return PrintOpenAPIError(cmd, err)
			}

			_, _ = fmt.Fprintln(h.verboseErrWriter, "Project created successfully!")
			cmdx.PrintRow(cmd, (*outputProject)(p))
			return nil
		},
	}

	cmd.Flags().StringP("name", "n", "", "The name of the project, required when quiet mode is used")
	cmdx.RegisterFormatFlags(cmd.Flags())
	return cmd
}
