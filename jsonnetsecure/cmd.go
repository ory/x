// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package jsonnetsecure

import (
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewJsonnetCmd() *cobra.Command {
	var (
		params processParameters
	)
	cmd := &cobra.Command{
		Use:    "jsonnet",
		Short:  "Run Jsonnet as a CLI command",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := params.DecodeFrom(cmd.InOrStdin()); err != nil {
				return err
			}
			vm := MakeSecureVM()

			for _, it := range params.ExtCodes {
				vm.ExtCode(it.Key, it.Value)
			}
			for _, it := range params.ExtVars {
				vm.ExtVar(it.Key, it.Value)
			}
			for _, it := range params.TLACodes {
				vm.TLACode(it.Key, it.Value)
			}
			for _, it := range params.TLAVars {
				vm.TLAVar(it.Key, it.Value)
			}

			result, err := vm.EvaluateAnonymousSnippet(params.Filename, params.Snippet)
			if err != nil {
				return errors.Wrap(err, "failed to evaluate snippet")
			}

			if _, err := io.WriteString(cmd.OutOrStdout(), result); err != nil {
				return errors.Wrap(err, "failed to write to stdout")
			}

			return nil
		},
	}

	return cmd
}
