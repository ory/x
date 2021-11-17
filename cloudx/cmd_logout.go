package cloudx

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewAuthLogoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Sign out of Ory Cloud",
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := NewSnakeCharmer(cmd)
			if err != nil {
				return err
			}
			if err := h.SignOut(); err != nil {
				return err
			}
			fmt.Println("You signed out successfully.")
			return nil
		},
	}
	RegisterFlags(cmd.PersistentFlags())
	return cmd
}
