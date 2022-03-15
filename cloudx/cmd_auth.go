package cloudx

import (
	"github.com/spf13/cobra"

	"github.com/ory/x/cmdx"
)

func NewAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Create an or sign into your Ory Cloud account",
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := NewSnakeCharmer(cmd)
			if err != nil {
				return err
			}
			ac, err := h.Authenticate()
			if err != nil {
				return err
			}
			cmdx.PrintRow(cmd, ac)
			return nil
		},
	}
	RegisterConfigFlag(cmd.PersistentFlags())
	RegisterYesFlag(cmd.PersistentFlags())
	cmdx.RegisterNoiseFlags(cmd.PersistentFlags())
	cmdx.RegisterFormatFlags(cmd.Flags())
	cmd.AddCommand(NewAuthLogoutCmd())
	return cmd
}
