package cmdx

import (
	"bytes"
	"text/template"

	"github.com/spf13/cobra"
)

// EnableUsageTemplating enables gotemplates for usage strings, i.e. cmd.Short, cmd.Long, and cmd.Example.
// The data for the template is the command itself. Especially useful are `.Root.Name` and `.CommandPath`.
// This will be inherited by all subcommands, so enabling it on the root command is sufficient.
func EnableUsageTemplating(cmd *cobra.Command) {
	cobra.AddTemplateFunc("insertTemplate", func(cmd *cobra.Command, tmpl string) (string, error) {
		t, err := template.New("").Parse(tmpl)
		if err != nil {
			return "", err
		}
		var out bytes.Buffer
		if err := t.Execute(&out, cmd); err != nil {
			return "", err
		}
		return out.String(), nil
	})
	cmd.SetHelpTemplate(`{{insertTemplate . (or .Long .Short) | trimTrailingWhitespaces}}

{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`)
	cmd.SetUsageTemplate(`Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{insertTemplate . .Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`)
}

// DisableUsageTemplating resets the commands usage template to the default.
// This can be used to undo the effects of EnableUsageTemplating, specifically for a subcommand.
func DisableUsageTemplating(cmd *cobra.Command) {
	defaultCmd := new(cobra.Command)
	cmd.SetHelpTemplate(defaultCmd.HelpTemplate())
	cmd.SetUsageTemplate(defaultCmd.UsageTemplate())
}
