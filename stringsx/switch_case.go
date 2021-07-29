package stringsx

import (
	"fmt"
	"strings"
)

type (
	RegisteredCases struct {
		cases  []string
		actual string
	}
	errUnknownCase struct {
		*RegisteredCases
	}
	RegisteredPrefixes struct {
		prefixes []string
		actual   string
	}
	errUnknownPrefix struct {
		*RegisteredPrefixes
	}
)

var (
	ErrUnknownCase   = errUnknownCase{}
	ErrUnknownPrefix = errUnknownPrefix{}
)

func SwitchExact(actual string) *RegisteredCases {
	return &RegisteredCases{
		actual: actual,
	}
}

func SwitchPrefix(actual string) *RegisteredPrefixes {
	return &RegisteredPrefixes{
		actual: actual,
	}
}

func (r *RegisteredCases) AddCase(c string) bool {
	r.cases = append(r.cases, c)
	return r.actual == c
}

func (r *RegisteredPrefixes) HasPrefix(prefix string) bool {
	r.prefixes = append(r.prefixes, prefix)
	return strings.HasPrefix(r.actual, prefix)
}

func (r *RegisteredCases) String() string {
	return "[" + strings.Join(r.cases, ", ") + "]"
}

func (r *RegisteredPrefixes) String() string {
	return "[" + strings.Join(r.prefixes, ", ") + "]"
}

func (r *RegisteredCases) ToUnknownCaseErr() error {
	return errUnknownCase{r}
}

func (r *RegisteredPrefixes) ToUnknownPrefixErr() error {
	return errUnknownPrefix{r}
}

func (e errUnknownCase) Error() string {
	return fmt.Sprintf("expected one of %s but got %s", e.String(), e.actual)
}

func (e errUnknownCase) Is(err error) bool {
	_, ok := err.(errUnknownCase)
	return ok
}

func (e errUnknownPrefix) Error() string {
	return fmt.Sprintf("expected %s to have one of the prefixes %s", e.actual, e.String())
}

func (e errUnknownPrefix) Is(err error) bool {
	_, ok := err.(errUnknownPrefix)
	return ok
}
