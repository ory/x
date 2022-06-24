package cmdx

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

// ParsePaginationArgs parses pagination arguments from the command line.
func ParsePaginationArgs(cmd *cobra.Command, pageArg, perPageArg string) (page, perPage int64, err error) {
	if len(pageArg+perPageArg) > 0 {
		page, err = strconv.ParseInt(pageArg, 0, 64)
		if err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Could not parse page argument\"%s\": %s", pageArg, err)
			return 0, 0, FailSilently(cmd)
		}

		perPage, err = strconv.ParseInt(perPageArg, 0, 64)
		if err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Could not parse per-page argument\"%s\": %s", perPageArg, err)
			return 0, 0, FailSilently(cmd)
		}
	}
	return
}

// ParseTokenPaginationArgs parses token-based pagination arguments from the command line.
func ParseTokenPaginationArgs(cmd *cobra.Command, pageArg, perPageArg string) (page string, perPage int64, err error) {
	if len(pageArg+perPageArg) > 0 {
		page = pageArg

		perPage, err = strconv.ParseInt(perPageArg, 0, 64)
		if err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Could not parse per-page argument\"%s\": %s", perPageArg, err)
			return "", 0, FailSilently(cmd)
		}
	}
	return
}
