package stringsx

import (
	"fmt"
	"strings"
)

type (
	RegisteredCases []string
	errUnknownCase  struct {
		cases  RegisteredCases
		actual string
	}
)

var ErrUnknownCase = errUnknownCase{}

func (r *RegisteredCases) AddCase(c string) string {
	*r = append(*r, c)
	return c
}

func (r *RegisteredCases) String() string {
	s := make([]string, len(*r))
	for i, c := range *r {
		s[i] = fmt.Sprintf("%#v", c)
	}
	return "[" + strings.Join(s, ", ") + "]"
}

func (r *RegisteredCases) ToUnknownCaseErr(actual string) error {
	return errUnknownCase{cases: *r, actual: actual}
}

func (e errUnknownCase) Error() string {
	return fmt.Sprintf("expected one of %s but got %s", e.cases.String(), e.actual)
}

func (e errUnknownCase) Is(err error) bool {
	_, ok := err.(errUnknownCase)
	return ok
}
