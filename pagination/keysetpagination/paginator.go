// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package keysetpagination

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/pop/v6/columns"
)

type (
	Item interface{ PageToken() string }

	Order string

	columnOrdering struct {
		name  string
		order Order
	}
	Paginator struct {
		token, defaultToken        string
		size, defaultSize, maxSize int
		isLast                     bool
		additionalColumn           columnOrdering
	}
	Option func(*Paginator) *Paginator
)

var ErrUnknownOrder = errors.New("unknown order")

const (
	OrderDescending Order = "DESC"
	OrderAscending  Order = "ASC"
)

func (o Order) extract() (string, string, error) {
	switch o {
	case OrderAscending:
		return ">", string(o), nil
	case OrderDescending:
		return "<", string(o), nil
	default:
		return "", "", ErrUnknownOrder
	}
}

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
		if columnName, value, found := strings.Cut(p, "="); found {
			r[columnName] = value
		}
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
		WithColumn(p.additionalColumn.name, p.additionalColumn.order),
		withIsLast(p.isLast),
	}
}

func (p *Paginator) multipleOrderFieldsQuery(q *pop.Query, idField string, cols map[string]*columns.Column, quote func(string) string) error {
	column, ok := cols[p.additionalColumn.name]
	if !ok {
		return errors.New("column not supported")
	}

	tokenParts := parseToken(idField, p.Token())
	idValue := tokenParts[idField]

	quoteName := quote(column.Name)

	value, ok := tokenParts[column.Name]

	if !ok {
		return errors.New("no value provided for " + column.Name)
	}

	sign, keyword, err := p.additionalColumn.order.extract()
	if err != nil {
		return err
	}

	q.
		Where(fmt.Sprintf("%s %s ? OR (%s = ? AND %s > ?)", quoteName, sign, quoteName, quote(idField)), value, value, idValue).
		Order(fmt.Sprintf("%s %s", quoteName, keyword))

	return nil
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

		if err := p.multipleOrderFieldsQuery(q, id, model.Columns().Cols, q.Connection.Dialect.Quote); err != nil {
			// silently ignore the error, and fall back to the "default" behavior of just ordering by the token
			q.Where(fmt.Sprintf(`%s > ?`, eid), p.Token())
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

func WithColumn(name string, order Order) Option {
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
