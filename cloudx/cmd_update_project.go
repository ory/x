package cloudx

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/ory/x/cmdx"

	"github.com/ory/x/flagx"
)

func NewProjectsUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project id",
		Args:  cobra.ExactArgs(1),
		Short: "Update Ory Cloud Project Service Configuration",
		Example: `$ ory update project ecaaa3cb-0730-4ee8-a6df-9553cdfeef89 \
	--name \"my updated name\" \
	--file /path/to/config.json \
	--file /path/to/config.yml \
	--file https://example.org/config.yaml \
	--file base64://<json>

ID		ecaaa3cb-0730-4ee8-a6df-9553cdfeef89 
SLUG	good-wright-t7kzy3vugf		
STATE	running					
NAME	Example Project	

$ ory update project ecaaa3cb-0730-4ee8-a6df-9553cdfeef89 \
	--name \"my updated name\" \
	--file /path/to/config.json \
	--format json-pretty

{
  "name": "my updated name",
  "identity": {
	"services": {
	  "config": {
		"courier": {
		  "smtp": {
			"from_name": "..."
		  }
		  // ...
		}
	  }
	}
  }
}`,
		Long: `Use this command to replace your current Ory Cloud Project's service configuration. All values
will be overwritten. To update individual settings use the ` + "`patch`" + ` command instead.

If the ` + "`--name`" + ` flag is not set, the project's name will not be changed.

The full configuration payload can be found at

	https://www.ory.sh/docs/reference/api#operation/updateProject

As an example an input could look like:

	{
      "name": "my updated name",
	  "identity": {
		"services": {
		  "config": {
			"courier": {
			  "smtp": {
				"from_name": "..."
			  }
			  // ...
			}
		  }
		}
	  }
	}
`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			h, err := NewSnakeCharmer(cmd)
			if err != nil {
				return err
			}

			name := flagx.MustGetString(cmd, "name")
			files := flagx.MustGetStringSlice(cmd, "file")
			if len(files) == 0 {
				return errors.New("--file must be set")
			}

			configs, err := ReadConfigFiles(files)
			if err != nil {
				return err
			}

			p, err := h.UpdateProject(args[0], name, configs)
			if err != nil {
				return PrintOpenAPIError(cmd, err)
			}

			cmdx.PrintRow(cmd, (*outputProject)(&p.Project))
			return h.PrintUpdateProjectWarnings(p)
		},
	}

	cmd.Flags().StringP("name", "n", "", "The name of the project, required when quiet mode is used")
	cmd.Flags().StringSliceP("file", "f", nil, "Configuration file(s) (file://config.json, https://example.org/config.yaml, ...) to update the project")
	RegisterYesFlag(cmd.Flags())
	cmdx.RegisterFormatFlags(cmd.Flags())
	return cmd
}
