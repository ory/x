package cmdx

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterFlags(t *testing.T) {
	setup := func() *pflag.FlagSet {
		flags := pflag.NewFlagSet("test flags", pflag.ContinueOnError)
		RegisterFormatFlags(flags)
		return flags
	}

	t.Run("case=format flags", func(t *testing.T) {
		t.Run("format=no value", func(t *testing.T) {
			flags := setup()
			require.NoError(t, flags.Parse([]string{}))
			f, err := flags.GetString(FlagFormat)
			require.NoError(t, err)

			assert.Equal(t, FormatDefault, format(f))
		})
	})
}
