// Copyright © 2025 Ory Corp
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package jsonnetsecure

func SetVirtualMemoryLimit(limit uint64) error {
	// TODO No-op for now. Apparently there is a Windows-specific equivalent (Job control)?
	return nil
}
