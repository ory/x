// Copyright © 2025 Ory Corp
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package jsonnetsecure

import (
	"fmt"
	"syscall"

	"github.com/pkg/errors"
)

func SetVirtualMemoryLimit(limitBytes uint64) error {
	lim := syscall.Rlimit{
		Cur: limitBytes,
		Max: limitBytes,
	}
	err := syscall.Setrlimit(syscall.RLIMIT_AS, &lim)
	if err != nil {
		return errors.WithStack(fmt.Errorf("failed to set virtual memory limit: %v\n", err))
	}
	return nil
}
