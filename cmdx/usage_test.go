package cmdx

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestUsageTemplating(t *testing.T) {
	root := &cobra.Command{
		Use:   "root",
		Short: "{{ .Name }}",
	}
	cmdWithTemplate := &cobra.Command{
		Use:     "with-template",
		Long:    "{{ .Name }}",
		Example: "{{ .Name }}",
	}
	cmdWithoutTemplate := &cobra.Command{
		Use:     "without-template",
		Long:    "{{ .Name }}",
		Example: "{{ .Name }}",
	}
	root.AddCommand(cmdWithTemplate, cmdWithoutTemplate)

	EnableUsageTemplating(root)
	DisableUsageTemplating(cmdWithoutTemplate)
	assert.NotContains(t, root.UsageString(), "{{ .Name }}")
	assert.NotContains(t, cmdWithTemplate.UsageString(), "{{ .Name }}")
	assert.Contains(t, cmdWithoutTemplate.UsageString(), "{{ .Name }}")
}
