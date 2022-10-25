package keysetpagination

import (
	"fmt"

	"github.com/gobuffalo/pop/v6"
)

type (
	Item      interface{ PageToken() string }
	Paginator struct {
		token, defaultToken        string
		size, defaultSize, maxSize int
		isLast                     bool
		sortDescending             bool
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

func (p *Paginator) IsLast() bool {
	return p.isLast
}

func (p *Paginator) SortDirection() string {
	if p.sortDescending {
		return "desc"
	}
	return "asc"
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
	id := (&pop.Model{Value: item}).IDField()
	return func(q *pop.Query) *pop.Query {
		return q.
			Limit(p.Size()+1).
			Where(fmt.Sprintf(`%q > ?`, id), p.Token()).
			Order(fmt.Sprintf(`%q ASC`, id))
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

func withIsLast(isLast bool) Option {
	return func(opts *Paginator) *Paginator {
		opts.isLast = isLast
		return opts
	}
}

func WithDescendingSort(descending bool) Option {
	return func(opts *Paginator) *Paginator {
		opts.sortDescending = descending
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
