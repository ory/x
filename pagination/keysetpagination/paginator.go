// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package keysetpagination

import (
	"errors"
	"fmt"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/pop/v6/columns"
)

type (
	Item interface{ PageToken() PageToken }

	Order string

	columnOrdering struct {
		name  string
		order Order
	}
	Paginator struct {
		token, defaultToken        PageToken
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

func (p *Paginator) Token() PageToken {
	if p.token == nil {
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

func (p *Paginator) multipleOrderFieldsQuery(q *pop.Query, idField string, cols map[string]*columns.Column, quote func(string) string) {
	tokenParts := p.Token().Parse(idField)
	idValue := tokenParts[idField]

	column, ok := cols[p.additionalColumn.name]
	if !ok {
		q.Where(fmt.Sprintf(`%s > ?`, quote(idField)), idValue)
		return
	}

	quoteName := quote(column.Name)

	value, ok := tokenParts[column.Name]

	if !ok {
		q.Where(fmt.Sprintf(`%s > ?`, quote(idField)), idValue)
		return
	}

	sign, keyword, err := p.additionalColumn.order.extract()
	if err != nil {
		q.Where(fmt.Sprintf(`%s > ?`, quote(idField)), idValue)
		return
	}

	q.
		Where(fmt.Sprintf("(%s %s ? OR (%s = ? AND %s > ?))", quoteName, sign, quoteName, quote(idField)), value, value, idValue).
		Order(fmt.Sprintf("%s %s", quoteName, keyword))

}

// Paginate returns a function that paginates a pop.Query.
// Usage:
//
//	q := c.Where("foo = ?", foo).Scope(keysetpagination.Paginate[MyItemType](paginator))
//
// This function works regardless of whether your type implements the Item
// interface with pointer or value receivers. To understand the type parameters,
// see this document:
// https://go.googlesource.com/proposal/+/refs/heads/master/design/43651-type-parameters.md#pointer-method-example
func Paginate[I any, PI interface {
	Item
	*I
}](p *Paginator) pop.ScopeFunc {
	model := pop.Model{Value: new(I)}
	id := model.IDField()
	return func(q *pop.Query) *pop.Query {
		eid := q.Connection.Dialect.Quote(id)

		p.multipleOrderFieldsQuery(q, id, model.Columns().Cols, q.Connection.Dialect.Quote)

		return q.
			Limit(p.Size() + 1).
			// we always need to order by the id field last
			Order(fmt.Sprintf(`%s ASC`, eid))
	}
}

// Result removes the last item (if applicable) and returns the paginator for the next page.
//
// This function works regardless of whether your type implements the Item
// interface with pointer or value receivers. To understand the type parameters,
// see this document:
// https://go.googlesource.com/proposal/+/refs/heads/master/design/43651-type-parameters.md#pointer-method-example
func Result[I any, PI interface {
	Item
	*I
}](items []I, p *Paginator) ([]I, *Paginator) {
	if len(items) > p.Size() {
		items = items[:p.Size()]
		return items, &Paginator{
			token:        PI(&items[len(items)-1]).PageToken(),
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

func WithDefaultToken(t PageToken) Option {
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

func WithToken(t PageToken) Option {
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
