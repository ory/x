package cmdx

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/ghodss/yaml"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type (
	TableHeader interface {
		Header() []string
	}
	TableRow interface {
		TableHeader
		Columns() []string
		Interface() interface{}
	}
	Table interface {
		TableHeader
		Table() [][]string
		Interface() interface{}
		Len() int
	}
	Nil struct{}

	format string
)

const (
	FormatQuiet      format = "quiet"
	FormatTable      format = "table"
	FormatJSON       format = "json"
	FormatYAML       format = "yaml"
	FormatJSONPretty format = "json-pretty"
	FormatDefault    format = "default"

	FlagFormat = "format"

	None = "<none>"
)

func (Nil) String() string {
	return "null"
}

func (Nil) Interface() interface{} {
	return nil
}

func PrintErrors(cmd *cobra.Command, errs map[string]error) {
	for src, err := range errs {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%s: %s\n", src, err.Error())
	}
}

func PrintRow(cmd *cobra.Command, row TableRow) {
	f := getFormat(cmd)

	switch f {
	case FormatQuiet:
		if idAble, ok := row.(interface{ ID() string }); ok {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), idAble.ID())
			break
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), row.Columns()[0])
	case FormatJSON:
		printJSON(cmd.OutOrStdout(), row.Interface(), false)
	case FormatYAML:
		printYAML(cmd.OutOrStdout(), row.Interface())
	case FormatJSONPretty:
		printJSON(cmd.OutOrStdout(), row.Interface(), true)
	case FormatTable, FormatDefault:
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 8, 1, '\t', 0)

		fields := row.Columns()
		for i, h := range row.Header() {
			_, _ = fmt.Fprintf(w, "%s\t%s\t\n", h, fields[i])
		}

		_ = w.Flush()
	}
}

func PrintTable(cmd *cobra.Command, table Table) {
	f := getFormat(cmd)

	switch f {
	case FormatQuiet:
		if table.Len() == 0 {
			fmt.Fprintln(cmd.OutOrStdout())
		}

		if idAble, ok := table.(interface{ IDs() []string }); ok {
			for _, row := range idAble.IDs() {
				fmt.Fprintln(cmd.OutOrStdout(), row)
			}
			break
		}

		for _, row := range table.Table() {
			fmt.Fprintln(cmd.OutOrStdout(), row[0])
		}
	case FormatJSON:
		printJSON(cmd.OutOrStdout(), table.Interface(), false)
	case FormatJSONPretty:
		printJSON(cmd.OutOrStdout(), table.Interface(), true)
	case FormatYAML:
		printYAML(cmd.OutOrStdout(), table.Interface())
	default:
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 8, 1, '\t', 0)

		for _, h := range table.Header() {
			fmt.Fprintf(w, "%s\t", h)
		}
		fmt.Fprintln(w)

		for _, row := range table.Table() {
			fmt.Fprintln(w, strings.Join(row, "\t")+"\t")
		}

		w.Flush()
	}
}

type interfacer interface{ Interface() interface{} }

func PrintJSONAble(cmd *cobra.Command, d interface{ String() string }) {
	if d == nil {
		d = Nil{}
	}
	switch getFormat(cmd) {
	default:
		_, _ = fmt.Fprint(cmd.OutOrStdout(), d.String())
	case FormatJSON:
		var v interface{} = d
		if i, ok := d.(interfacer); ok {
			v = i
		}
		printJSON(cmd.OutOrStdout(), v, false)
	case FormatJSONPretty:
		var v interface{} = d
		if i, ok := d.(interfacer); ok {
			v = i
		}
		printJSON(cmd.OutOrStdout(), v, true)
	case FormatYAML:
		var v interface{} = d
		if i, ok := d.(interfacer); ok {
			v = i
		}
		printJSON(cmd.OutOrStdout(), v, true)
	}
}

func getQuiet(cmd *cobra.Command) bool {
	q, err := cmd.Flags().GetBool(FlagQuiet)
	// ignore the error here as we use this function also when the flag might not be registered
	if err != nil {
		return false
	}
	return q
}

func getFormat(cmd *cobra.Command) format {
	q := getQuiet(cmd)

	if q {
		return FormatQuiet
	}

	f, err := cmd.Flags().GetString(FlagFormat)
	// unexpected error
	Must(err, "flag access error: %s", err)

	switch f {
	case string(FormatTable):
		return FormatTable
	case string(FormatJSON):
		return FormatJSON
	case string(FormatJSONPretty):
		return FormatJSONPretty
	case string(FormatYAML):
		return FormatYAML
	default:
		return FormatDefault
	}
}

func printJSON(w io.Writer, v interface{}, pretty bool) {
	e := json.NewEncoder(w)
	if pretty {
		e.SetIndent("", "  ")
	}
	err := e.Encode(v)
	// unexpected error
	Must(err, "Error encoding JSON: %s", err)
}

func printYAML(w io.Writer, v interface{}) {
	e, err := yaml.Marshal(v)
	Must(err, "Error encoding YAML: %s", err)
	_, _ = w.Write(e)
}

func RegisterJSONFormatFlags(flags *pflag.FlagSet) {
	flags.String(FlagFormat, string(FormatDefault), fmt.Sprintf("Set the output format. One of %s, %s, %s, and %s.", FormatDefault, FormatJSON, FormatYAML, FormatJSONPretty))
}

func RegisterFormatFlags(flags *pflag.FlagSet) {
	RegisterNoiseFlags(flags)
	flags.String(FlagFormat, string(FormatDefault), fmt.Sprintf("Set the output format. One of %s, %s, %s, and %s.", FormatTable, FormatJSON, FormatYAML, FormatJSONPretty))
}
