package dbal

import (
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	drivers = make(map[string]Driver)
	dmtx    sync.Mutex
)

// Driver represents a driver
type Driver interface {
	// Ping returns nil if the driver is alive or an error otherwise.
	Ping() error

	// Schemes returns a list of schemes this driver supports (e.g. mysql, postgres).
	Schemes() []string

	// Init is used to initialize the driver (e.g. connect to the datastore).
	Init(url string, l logrus.FieldLogger, opts ...DriverOptionModifier) error
}

// RegisterDriver registers a driver.
func RegisterDriver(b Driver) {
	dmtx.Lock()
	for _, prefix := range b.Schemes() {
		drivers[prefix] = b
	}
	dmtx.Unlock()
}

// RegisteredDriverSchemes returns the registered driver schemes.
func RegisteredDriverSchemes() []string {
	keys := make([]string, len(drivers))
	i := 0
	for k := range drivers {
		keys[i] = k
		i++
	}
	return keys
}
