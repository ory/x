package cloudx

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: fmt.Sprintf("Update resources"),
	}
	cmd.AddCommand(NewProjectsUpdateCmd())
	RegisterConfigFlag(cmd.PersistentFlags())
	return cmd
}
