package cloudx

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewPatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "patch",
		Short: fmt.Sprintf("Patch resources"),
	}
	RegisterConfigFlag(cmd.PersistentFlags())
	cmd.AddCommand(NewProjectsPatchCmd())
	cmd.AddCommand(NewPatchKratosConfigCmd())
	return cmd
}
