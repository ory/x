// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package jsonnetx

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/bmatcuk/doublestar/v2"
	"github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/linter"
	"github.com/spf13/cobra"

	"github.com/ory/x/flagx"

	"github.com/ory/x/cmdx"
)

// LintCommand represents the lint command
var LintCommand = &cobra.Command{
	Use: "lint path/to/files/*.jsonnet [more/files.jsonnet, [supports/**/{foo,bar}.jsonnet]]",
	Long: `Lints JSONNet files using the official JSONNet linter and exits with a status code of 1 when issues are detected.

` + GlobHelp,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		verbose := flagx.MustGetBool(cmd, "verbose")
		for _, pattern := range args {
			files, err := doublestar.Glob(pattern)
			cmdx.Must(err, `Glob path "%s" is not valid: %s`, pattern, err)

			for _, file := range files {
				if fi, err := os.Stat(file); err != nil {
					cmdx.Must(err, "Unable to stat file %s: %s", file, err)
				} else if fi.IsDir() {
					continue
				}

				if verbose {
					fmt.Printf("Processing file: %s\n", file)
				}

				//#nosec G304 -- false positive
				content, err := ioutil.ReadFile(file)
				cmdx.Must(err, `Unable to read file "%s" because: %s`, file, err)

				if linter.LintSnippet(jsonnet.MakeVM(), os.Stderr, []linter.Snippet{{FileName: file, Code: string(content)}}) {
					_, _ = fmt.Fprintf(os.Stderr, "Linter found issues.")
					os.Exit(1)
				}
			}
		}
	},
}

func init() {
	LintCommand.Flags().Bool("verbose", false, "Verbose output.")
}
