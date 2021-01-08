package cmdx

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type conditionalPrinter struct {
	w     io.Writer
	print bool
}

const (
	FlagQuiet = "quiet"
)

func RegisterNoiseFlags(flags *pflag.FlagSet) {
	flags.BoolP(FlagQuiet, FlagQuiet[:1], false, "Be quiet with output printing.")
}

// NewLoudOutPrinter returns a conditionalPrinter that
// only prints to cmd.OutOrStdout when --quiet is not set
func NewLoudOutPrinter(cmd *cobra.Command) *conditionalPrinter {
	quiet, err := cmd.Flags().GetBool(FlagQuiet)
	if err != nil {
		Fatalf(err.Error())
	}

	return &conditionalPrinter{
		w:     cmd.OutOrStdout(),
		print: !quiet,
	}
}

// NewQuietOutPrinter returns a conditionalPrinter that
// only prints to cmd.OutOrStdout when --quiet is set
func NewQuietOutPrinter(cmd *cobra.Command) *conditionalPrinter {
	quiet, err := cmd.Flags().GetBool(FlagQuiet)
	if err != nil {
		Fatalf(err.Error())
	}

	return &conditionalPrinter{
		w:     cmd.OutOrStdout(),
		print: quiet,
	}
}

// NewLoudErrPrinter returns a conditionalPrinter that
// only prints to cmd.ErrOrStderr when --quiet is not set
func NewLoudErrPrinter(cmd *cobra.Command) *conditionalPrinter {
	quiet, err := cmd.Flags().GetBool(FlagQuiet)
	if err != nil {
		Fatalf(err.Error())
	}

	return &conditionalPrinter{
		w:     cmd.ErrOrStderr(),
		print: !quiet,
	}
}

// NewQuietErrPrinter returns a conditionalPrinter that
// only prints to cmd.ErrOrStderr when --quiet is set
func NewQuietErrPrinter(cmd *cobra.Command) *conditionalPrinter {
	quiet, err := cmd.Flags().GetBool(FlagQuiet)
	if err != nil {
		Fatalf(err.Error())
	}

	return &conditionalPrinter{
		w:     cmd.ErrOrStderr(),
		print: quiet,
	}
}

// NewLoudPrinter returns a conditionalPrinter that
// only prints to w when --quiet is not set
func NewLoudPrinter(cmd *cobra.Command, w io.Writer) *conditionalPrinter {
	quiet, err := cmd.Flags().GetBool(FlagQuiet)
	if err != nil {
		Fatalf(err.Error())
	}

	return &conditionalPrinter{
		w:     w,
		print: !quiet,
	}
}

// NewQuietPrinter returns a conditionalPrinter that
// only prints to w when --quiet is set
func NewQuietPrinter(cmd *cobra.Command, w io.Writer) *conditionalPrinter {
	quiet, err := cmd.Flags().GetBool(FlagQuiet)
	if err != nil {
		Fatalf(err.Error())
	}

	return &conditionalPrinter{
		w:     w,
		print: quiet,
	}
}

func NewConditionalPrinter(w io.Writer, print bool) *conditionalPrinter {
	return &conditionalPrinter{
		w:     w,
		print: print,
	}
}

func (p *conditionalPrinter) Println(a ...interface{}) (n int, err error) {
	if p.print {
		return fmt.Fprintln(p.w, a...)
	}
	return
}

func (p *conditionalPrinter) Print(a ...interface{}) (n int, err error) {
	if p.print {
		return fmt.Fprint(p.w, a...)
	}
	return
}

func (p *conditionalPrinter) Printf(format string, a ...interface{}) (n int, err error) {
	if p.print {
		return fmt.Fprintf(p.w, format, a...)
	}
	return
}
