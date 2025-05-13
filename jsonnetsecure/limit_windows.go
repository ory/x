//go:build windows

package jsonnetsecure

func SetVirtualMemoryLimit(limit uint64) error {
	// No-op for now. Apparently there is a Windows-specific equivalent (Job control)?
	return nil
}
