//go:build !windows

package jsonnetsecure

import (
	"fmt"
	"syscall"

	"github.com/pkg/errors"
)

func SetVirtualMemoryLimit(limit uint64) error {
	lim := syscall.Rlimit{
		Cur: limit,
		Max: limit,
	}
	err := syscall.Setrlimit(syscall.RLIMIT_AS, &lim)
	if err != nil {
		return errors.WithStack(fmt.Errorf("failed to set virtual memory limit: %v\n", err))
	}
	return nil
}
