package vapi

import (
	"github.com/Masterminds/semver/v3"
	"github.com/julienschmidt/httprouter"
)

type Manager struct{}

type managerOptions struct {
	versions map[semver.Version]Migrator
}

type ManagerOption func(*managerOptions)

func newManagerOptions() *managerOptions {
	return &managerOptions{

	}
}

func (o *managerOptions) apply(opts []ManagerOption) *managerOptions {
	for _, f := range opts {
		f(o)
	}
	return o
}

func (m *Manager) WrapHttpRouterHandle(inner httprouter.Handle, opts ...ManagerOption) {
	_ = newManagerOptions().apply(opts)
}
