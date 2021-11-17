package cloudx

import (
	"github.com/spf13/cobra"
)

func NewAuthCmd(self string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Create or sign into your Ory Cloud account",
		RunE: func(cmd *cobra.Command, args []string) error {
			h, err := NewSnakeCharmer(cmd)
			if err != nil {
				return err
			}
			if _, err = h.Authenticate(); err != nil {
				return err
			}
			return nil
		},
	}
	RegisterFlags(cmd.PersistentFlags())
	return cmd
}
