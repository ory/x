// Copyright © 2023 Ory Corp
// SPDX-License-Identifier: Apache-2.0

package sqlxx

import (
	"context"
	"sort"
	"strings"

	"github.com/gobuffalo/pop/v6"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// Expandable controls what fields to expand for projects.
type Expandable string

// Expandables is a list of Expandable values.
type Expandables []Expandable

// String returns a string representation of the Expandable.
func (e Expandable) String() string {
	return string(e)
}

// ToEager returns the fields used by pop's Eager command.
func (e Expandables) ToEager() []string {
	var s []string
	for _, e := range e {
		s = append(s, e.String())
	}
	return s
}

// Has returns true if the Expandable is in the list.
func (e Expandables) Has(search Expandable) bool {
	for _, e := range e {
		if e == search {
			return true
		}
	}
	return false
}

func (e Expandables) Load(ctx context.Context, c *pop.Connection, m interface{}) error {
	e.Sort()
	return e.load(ctx, c, m, 1, e.MaxLevels())
}

func (e Expandables) load(ctx context.Context, c *pop.Connection, m interface{}, level int, maxLevel int) error {
	var eg errgroup.Group
	for i := range e {
		item := e[i].String()
		if len(strings.Split(item, ".")) == level {
			continue
		}

		eg.Go(func() error {
			// We need a copy of the connection which is only possible using `WithContext` because
			// `.clone()` is not exported.
			return errors.WithStack(c.WithContext(ctx).Load(m, item))
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	if level <= maxLevel {
		return e.load(ctx, c, m, level+1, maxLevel)
	}

	return nil
}

func (e Expandables) Sort() {
	sort.SliceStable(e, func(i, j int) bool {
		if len(strings.Split(e[i].String(), ".")) < len(strings.Split(e[j].String(), ".")) {
			return true
		}
		return e[i] < e[j]
	})
}

func (e Expandables) MaxLevels() (max int) {
	for _, e := range e {
		if l := len(strings.Split(e.String(), ".")); l > max {
			max = l
		}
	}
	return
}
