// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package jsonnetx

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/bmatcuk/doublestar/v2"
	"github.com/google/go-jsonnet/formatter"
	"github.com/spf13/cobra"

	"github.com/ory/x/cmdx"
	"github.com/ory/x/flagx"
)

// FormatCommand represents the format command
var FormatCommand = &cobra.Command{
	Use: "format path/to/files/*.jsonnet [more/files.jsonnet, [supports/**/{foo,bar}.jsonnet]]",
	Long: `Formats JSONNet files using the official JSONNet formatter.

Use -w or --write to write output back to files instead of stdout.

` + GlobHelp,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		verbose := flagx.MustGetBool(cmd, "verbose")
		for _, pattern := range args {
			files, err := doublestar.Glob(pattern)
			cmdx.Must(err, `Glob pattern "%s" is not valid: %s`, pattern, err)

			shouldWrite := flagx.MustGetBool(cmd, "write")
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

				output, err := formatter.Format(file, string(content), formatter.DefaultOptions())
				cmdx.Must(err, `JSONNet file "%s" could not be formatted: %s`, file, err)

				if shouldWrite {
					err := ioutil.WriteFile(file, []byte(output), 0644) // #nosec
					cmdx.Must(err, `Could not write to file "%s" because: %s`, file, err)
				} else {
					fmt.Println(output)
				}
			}
		}
	},
}

func init() {
	FormatCommand.Flags().BoolP("write", "w", false, "Write formatted output back to file.")
	FormatCommand.Flags().Bool("verbose", false, "Verbose output.")
}
