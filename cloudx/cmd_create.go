package cloudx

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ory/x/cmdx"
)

func NewCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: fmt.Sprintf("Create Ory Cloud resources"),
	}
	cmd.AddCommand(NewCreateProjectCmd())
	RegisterConfigFlag(cmd.PersistentFlags())
	RegisterYesFlag(cmd.PersistentFlags())
	cmdx.RegisterNoiseFlags(cmd.PersistentFlags())
	cmdx.RegisterJSONFormatFlags(cmd.PersistentFlags())
	return cmd
}
