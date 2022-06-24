package cmdx

type (
	// OutputIder outputs an ID
	OutputIder string
	// OutputIderCollection outputs a list of IDs
	OutputIderCollection struct {
		Items []OutputIder
	}
)

func (OutputIder) Header() []string {
	return []string{"ID"}
}

func (i OutputIder) Columns() []string {
	return []string{string(i)}
}

func (i OutputIder) Interface() interface{} {
	return i
}

func (OutputIderCollection) Header() []string {
	return []string{"ID"}
}

func (c OutputIderCollection) Table() [][]string {
	rows := make([][]string, len(c.Items))
	for i, ident := range c.Items {
		rows[i] = []string{string(ident)}
	}
	return rows
}

func (c OutputIderCollection) Interface() interface{} {
	return c.Items
}

func (c OutputIderCollection) Len() int {
	return len(c.Items)
}
