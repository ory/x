package cmdx

import (
	"io"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPagination(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetErr(io.Discard)
	page, perPage, err := ParsePaginationArgs(cmd, "1", "2")
	require.NoError(t, err)
	assert.EqualValues(t, 1, page)
	assert.EqualValues(t, 2, perPage)

	_, _, err = ParsePaginationArgs(cmd, "abcd", "")
	require.Error(t, err)
}
