package cloudx

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewGetProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project id",
		Args:  cobra.ExactArgs(1),
		Short: fmt.Sprintf("Get an Ory Cloud project"),
		Example: `ory get project ecaaa3cb-0730-4ee8-a6df-9553cdfeef89
ory get project ecaaa3cb-0730-4ee8-a6df-9553cdfeef89 --format json
ory get project ecaaa3cb-0730-4ee8-a6df-9553cdfeef89 --format kratos-config > kratos-config.yml`,
		Long: `If you wish to generate a configuration for self-hosting Ory Kratos, use ` + "`--format kratos-config`" + `.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := NewSnakeCharmer(cmd)
			if err != nil {
				return err
			}

			project, err := h.GetProject(args[0])
			if err != nil {
				return PrintOpenAPIError(cmd, err)
			}

			return PrintExtendedFormat(cmd, project)
		},
	}

	RegisterExtendedOutput(cmd.Flags())
	return cmd
}
