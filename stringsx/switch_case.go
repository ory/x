package stringsx

import (
	"fmt"
	"strings"
)

type (
	registeredCases struct {
		cases  []string
		actual string
	}
	errUnknownCase struct {
		*registeredCases
	}
	registeredPrefixes struct {
		prefixes []string
		actual   string
	}
	errUnknownPrefix struct {
		*registeredPrefixes
	}
)

var (
	ErrUnknownCase   = errUnknownCase{}
	ErrUnknownPrefix = errUnknownPrefix{}
)

func SwitchExact(actual string) *registeredCases {
	return &registeredCases{
		actual: actual,
	}
}

func SwitchPrefix(actual string) *registeredPrefixes {
	return &registeredPrefixes{
		actual: actual,
	}
}

func (r *registeredCases) AddCase(c string) bool {
	r.cases = append(r.cases, c)
	return r.actual == c
}

func (r *registeredPrefixes) HasPrefix(prefix string) bool {
	r.prefixes = append(r.prefixes, prefix)
	return strings.HasPrefix(r.actual, prefix)
}

func (r *registeredCases) String() string {
	return "[" + strings.Join(r.cases, ", ") + "]"
}

func (r *registeredPrefixes) String() string {
	return "[" + strings.Join(r.prefixes, ", ") + "]"
}

func (r *registeredCases) ToUnknownCaseErr() error {
	return errUnknownCase{r}
}

func (r *registeredPrefixes) ToUnknownPrefixErr() error {
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
