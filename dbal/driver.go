package dbal

import (
	"github.com/sirupsen/logrus"
	"sync"
)

var (
	drivers  = make(map[string]Driver)
	dmtx    sync.Mutex
)

// Driver represents a driver
type Driver interface {
	Ping() error
	Schemes() []string
	Init(url string, l logrus.FieldLogger, opts ...DriverOptionModifier) error
}

// RegisterDriver registers a driver
func RegisterDriver(b Driver) {
	dmtx.Lock()
	for _, prefix := range b.Schemes() {
		drivers[prefix] = b
	}
	dmtx.Unlock()
}

// RegisteredDrivers returns the registered driver schemes.
func RegisteredDriverSchemes() []string {
	keys := make([]string, len(drivers))
	i := 0
	for k := range drivers {
		keys[i] = k
		i++
	}
	return keys
}
