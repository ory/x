package cloudx

import (
	"github.com/ory/x/cmdx"

	"github.com/spf13/cobra"
)

func NewGetKratosConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "identity-config project-id",
		Aliases: []string{"ic", "kratos-config"},
		Args:    cobra.ExactArgs(1),
		Short:   "Get an Ory Cloud project's identity configuration",
		Long:    "You can use this command to render Ory Kratos configurations as well.",
		Example: `$ ory get kratos-config ecaaa3cb-0730-4ee8-a6df-9553cdfeef89 --format yaml > kratos-config.yaml

$ ory get kratos-config ecaaa3cb-0730-4ee8-a6df-9553cdfeef89 --format json

{
  "selfservice": {
	"methods": {
	  "password": { "enabled": false }
	}
	// ...
  }
}`,
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := NewSnakeCharmer(cmd)
			if err != nil {
				return err
			}

			project, err := h.GetProject(args[0])
			if err != nil {
				return PrintOpenAPIError(cmd, err)
			}

			cmdx.PrintJSONAble(cmd, outputConfig(project.Services.Identity.Config))
			return nil
		},
	}

	cmdx.RegisterJSONFormatFlags(cmd.Flags())
	return cmd
}
