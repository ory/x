package cloudx

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func NewRootCommand(project string, version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cloud",
		Short: fmt.Sprintf("Run and manage Ory %s in Ory Cloud", project),
	}

	cmdName := strings.ToLower(project + " cloud")

	cmd.AddCommand(NewAuthCmd())
	cmd.AddCommand(NewAuthLogoutCmd())
	cmd.AddCommand(NewCreateCmd())
	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewPatchCmd())
	cmd.AddCommand(NewUpdateCmd())
	cmd.AddCommand(NewGetCmd())
	cmd.AddCommand(NewProxyCommand(cmdName, version))
	cmd.AddCommand(NewTunnelCommand(cmdName, version))
	return cmd
}
