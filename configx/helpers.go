package configx

import (
	"github.com/spf13/pflag"
)

// RegisterFlags registers the config file flag.
func RegisterFlags(flags *pflag.FlagSet) {
	flags.StringSliceP("config", "c", []string{}, "Path to one or more .json, .yaml, .yml, .toml config files. Values are loaded in the order provided, meaning that the last config file overwrites values from the previous config file.")
}
