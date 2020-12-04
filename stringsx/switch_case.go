package stringsx

type RegisteredCases []string

func (r *RegisteredCases) AddCase(c string) string {
	*r = append(*r, c)
	return c
}
