package jsonnet

import (
	"github.com/spf13/cobra"
)

const GlobHelp = `Glob Syntax:

    pattern:
        { term }

    term:
        '*'         matches any sequence of non-separator characters
        '**'        matches any sequence of characters
        '?'         matches any single non-separator character
        '[' [ '!' ] { character-range } ']'
                    character class (must be non-empty)
        '{' pattern-list '}'
                    pattern alternatives
        c           matches character c (c != '*', '**', '?', '\', '[', '{', '}')
        '\' c       matches character c

    character-range:
        c           matches character c (c != '\\', '-', ']')
        '\' c       matches character c
        lo '-' hi   matches character c for lo <= c <= hi

    pattern-list:
        pattern { ',' pattern }
                    comma-separated (without spaces) patterns`

// RootCommand represents the jsonnet command
var RootCommand = &cobra.Command{
	Use:   "jsonnet",
	Short: "Helpers for linting and formatting JSONNet code",
}

// RegisterCommandRecursive adds all jsonnet helpers to the RootCommand
func RegisterCommandRecursive(parent *cobra.Command) {
	parent.AddCommand(RootCommand)

	RootCommand.AddCommand(FormatCommand)
	RootCommand.AddCommand(LintCommand)
}
