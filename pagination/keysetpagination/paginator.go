// Copyright © 2022 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package keysetpagination

import (
	"fmt"

	"github.com/gobuffalo/pop/v6"
)

type (
	Item      interface{ PageToken() string }
	Paginator struct {
		token, defaultToken        string
		orderByColumn              string
		orderDirection             string
		size, defaultSize, maxSize int
		isLast                     bool
	}
	Option         func(*Paginator) *Paginator
	OrderDirection string
)

const (
	OrderDirectionAscending  OrderDirection = "ASC"
	OrderDirectionDescending OrderDirection = "DESC"
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

func (p *Paginator) orderClause(pkColumn string, quoteFunc func(string) string) string {
	direction := string(OrderDirectionAscending)
	if p.orderDirection != "" {
		direction = p.orderDirection
	}

	if p.orderByColumn != "" {
		return fmt.Sprintf("%s %s", quoteFunc(p.orderByColumn), direction)
	}

	return fmt.Sprintf("%s %s", pkColumn, direction)
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
		WithOrder(p.orderByColumn, OrderDirection(p.orderDirection)),
	}
}

// Paginate returns a function that paginates a pop.Query.
// Usage:
//
//	q := c.Where("foo = ?", foo).Scope(keysetpagination.Paginate[Item](paginator))
func Paginate[I Item](p *Paginator) pop.ScopeFunc {
	var item I
	id := (&pop.Model{Value: item}).IDField()
	return func(q *pop.Query) *pop.Query {
		eid := q.Connection.Dialect.Quote(id)
		return q.
			Limit(p.Size()+1).
			Where(fmt.Sprintf(`%s > ?`, eid), p.Token()).
			Order(p.orderClause(eid, q.Connection.Dialect.Quote))
	}
}

// Result removes the last item (if applicable) and returns the paginator for the next page.
func Result[I Item](items []I, p *Paginator) ([]I, *Paginator) {
	if len(items) > p.Size() {
		items = items[:p.Size()]
		return items, &Paginator{
			token:          items[len(items)-1].PageToken(),
			defaultToken:   p.defaultToken,
			size:           p.size,
			defaultSize:    p.defaultSize,
			maxSize:        p.maxSize,
			orderByColumn:  p.orderByColumn,
			orderDirection: p.orderDirection,
		}
	}
	return items, &Paginator{
		defaultToken:   p.defaultToken,
		size:           p.size,
		defaultSize:    p.defaultSize,
		maxSize:        p.maxSize,
		isLast:         true,
		orderByColumn:  p.orderByColumn,
		orderDirection: p.orderDirection,
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

func WithOrder(column string, direction OrderDirection) Option {
	return func(opts *Paginator) *Paginator {
		opts.orderByColumn = column
		opts.orderDirection = string(direction)
		return opts
	}
}
