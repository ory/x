// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package keysetpagination

import (
	"fmt"
	"strings"

	"github.com/gobuffalo/pop/v6"
)

type (
	Item interface{ PageToken() string }

	columnOrdering struct {
		name  string
		order string
	}
	Paginator struct {
		token, defaultToken        string
		size, defaultSize, maxSize int
		isLast                     bool
		additionalColumn           columnOrdering
	}
	Option func(*Paginator) *Paginator
)

func (p *Paginator) Token() string {
	if p.token == "" {
		return p.defaultToken
	}
	return p.token
}

func (p *Paginator) Size() int {
	size := p.size
	if size == 0 {
		size = p.defaultSize
		if size == 0 {
			size = 100
		}
	}
	if p.maxSize > 0 && size > p.maxSize {
		size = p.maxSize
	}
	return size
}

func parseToken(idField string, s string) map[string]string {
	tokens := strings.Split(s, "/")
	if len(tokens) != 2 {
		return map[string]string{idField: s}
	}

	r := map[string]string{}

	for _, p := range tokens {
		parts := strings.Split(p, "=")
		if len(parts) != 2 {
			continue
		}
		r[parts[0]] = parts[1]
	}

	return r
}

func (p *Paginator) IsLast() bool {
	return p.isLast
}

func (p *Paginator) ToOptions() []Option {
	return []Option{
		WithToken(p.token),
		WithSize(p.size),
		WithDefaultToken(p.defaultToken),
		WithDefaultSize(p.defaultSize),
		WithMaxSize(p.maxSize),
		withIsLast(p.isLast),
	}
}

// Paginate returns a function that paginates a pop.Query.
// Usage:
//
//	q := c.Where("foo = ?", foo).Scope(keysetpagination.Paginate[Item](paginator))
func Paginate[I Item](p *Paginator) pop.ScopeFunc {
	var item I
	model := &pop.Model{Value: item}
	id := model.IDField()
	return func(q *pop.Query) *pop.Query {
		eid := q.Connection.Dialect.Quote(id)

		tokenParts := parseToken(id, p.Token())
		idValue := tokenParts[id]
		if column, ok := model.Columns().Cols[p.additionalColumn.name]; ok {
			quoteName := q.Connection.Dialect.Quote(column.Name)

			value := tokenParts[column.Name]

			q = q.
				Where(fmt.Sprintf("%s > ? OR (%s = ? AND %s > ?)", quoteName, quoteName, eid), value, value, idValue).
				Order(fmt.Sprintf("%s %s", quoteName, p.additionalColumn.order))
		} else {
			q = q.
				Where(fmt.Sprintf(`%s > ?`, eid), idValue)
		}
		return q.
			Limit(p.Size() + 1).
			// we always need to order by the id field last
			Order(fmt.Sprintf(`%s ASC`, eid))
	}
}

// Result removes the last item (if applicable) and returns the paginator for the next page.
func Result[I Item](items []I, p *Paginator) ([]I, *Paginator) {
	if len(items) > p.Size() {
		items = items[:p.Size()]
		return items, &Paginator{
			token:        items[len(items)-1].PageToken(),
			defaultToken: p.defaultToken,
			size:         p.size,
			defaultSize:  p.defaultSize,
			maxSize:      p.maxSize,
		}
	}
	return items, &Paginator{
		defaultToken: p.defaultToken,
		size:         p.size,
		defaultSize:  p.defaultSize,
		maxSize:      p.maxSize,
		isLast:       true,
	}
}

func WithDefaultToken(t string) Option {
	return func(opts *Paginator) *Paginator {
		opts.defaultToken = t
		return opts
	}
}

func WithDefaultSize(size int) Option {
	return func(opts *Paginator) *Paginator {
		opts.defaultSize = size
		return opts
	}
}

func WithMaxSize(size int) Option {
	return func(opts *Paginator) *Paginator {
		opts.maxSize = size
		return opts
	}
}

func WithToken(t string) Option {
	return func(opts *Paginator) *Paginator {
		opts.token = t
		return opts
	}
}

func WithSize(size int) Option {
	return func(opts *Paginator) *Paginator {
		opts.size = size
		return opts
	}
}

func WithColumn(name string, order string) Option {
	return func(opts *Paginator) *Paginator {
		opts.additionalColumn = columnOrdering{
			name:  name,
			order: order,
		}
		return opts
	}
}

func withIsLast(isLast bool) Option {
	return func(opts *Paginator) *Paginator {
		opts.isLast = isLast
		return opts
	}
}

func GetPaginator(modifiers ...Option) *Paginator {
	opts := &Paginator{}
	for _, f := range modifiers {
		opts = f(opts)
	}
	return opts
}
