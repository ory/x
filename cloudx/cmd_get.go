package cloudx

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ory/x/cmdx"
)

func NewGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: fmt.Sprintf("Get a resource"),
	}
	cmd.AddCommand(NewGetProjectCmd())
	cmd.AddCommand(NewGetKratosConfigCmd())
	RegisterConfigFlag(cmd.PersistentFlags())
	RegisterYesFlag(cmd.PersistentFlags())
	cmdx.RegisterNoiseFlags(cmd.PersistentFlags())
	return cmd
}
