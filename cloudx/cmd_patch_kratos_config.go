package cloudx

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/ory/x/cmdx"

	"github.com/ory/x/flagx"
)

func NewPatchKratosConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "identity-config project-id",
		Aliases: []string{"ic", "kratos-config"},
		Args:    cobra.ExactArgs(1),
		Short:   "Patch an Ory Cloud Project's Identity Config",
		Example: `$ ory patch project ecaaa3cb-0730-4ee8-a6df-9553cdfeef89 \
	--add '/courier/smtp={"from_name":"My new email name"}' \
	--replace '/selfservice/methods/password/enabled=false' \
	--delete '/selfservice/methods/totp/enabled' \
	--format json

{
  "selfservice": {
	"methods": {
	  "password": { "enabled": false }
	}
	// ...
  }
}`,
		Long: `Use this command to patch your current Ory Cloud Project's identity service configuration. Only values
specified in the patch will be overwritten. To replace the config use the ` + "`update`" + ` command instead.

Compared to the ` + "`patch project`" + ` command, this command only updates the identity service configuration
and also only returns the identity service configuration as a result. This command is useful when you want to
import an Ory Kratos config as well, for example. This allows for shorter paths when specifying the flags

	ory patch identity-config ecaaa3cb-0730-4ee8-a6df-9553cdfeef89 \
		--replace '/selfservice/methods/password/enabled=false'

when compared to the ` + "`patch project`" + ` command:

	ory patch project ecaaa3cb-0730-4ee8-a6df-9553cdfeef89 \
		--replace '/identity/services/config/selfservice/methods/password/enabled=false'

The format of the patch is a JSON-Patch document. For more details please check:

	https://www.ory.sh/docs/reference/api#operation/patchProject
	https://jsonpatch.com`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			h, err := NewSnakeCharmer(cmd)
			if err != nil {
				return err
			}

			files := flagx.MustGetStringSlice(cmd, "file")
			add := prefixIdentityConfig(flagx.MustGetStringArray(cmd, "add"))
			replace := prefixIdentityConfig(flagx.MustGetStringArray(cmd, "replace"))
			remove := prefixIdentityConfig(flagx.MustGetStringArray(cmd, "remove"))

			if len(files)+len(add)+len(replace)+len(remove) == 0 {
				return errors.New("at least one of --file, --add, --replace, or --remove must be set")
			}

			configs, err := ReadConfigFiles(files)
			if err != nil {
				return err
			}

			configs, err = prefixFileIdentityConfig(configs)
			if err != nil {
				return err
			}

			p, err := h.PatchProject(args[0], configs, add, replace, remove)
			if err != nil {
				return PrintOpenAPIError(cmd, err)
			}

			cmdx.PrintJSONAble(cmd, outputConfig(p.Project.Services.Identity.Config))
			return h.PrintUpdateProjectWarnings(p)
		},
	}

	cmd.Flags().StringSliceP("file", "f", nil, "Configuration file(s) (file://config.json, https://example.org/config.yaml, ...) to update the project")
	cmd.Flags().StringArray("replace", nil, "Replace a specific key in the configuration")
	cmd.Flags().StringArray("add", nil, "Add a specific key to the configuration")
	cmd.Flags().StringArray("remove", nil, "Remove a specific key from the configuration")
	RegisterYesFlag(cmd.Flags())
	cmdx.RegisterFormatFlags(cmd.Flags())
	return cmd
}
