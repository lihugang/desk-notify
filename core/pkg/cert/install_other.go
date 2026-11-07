//go:build !windows

package cert

// EnsureRootCAInstalled is a no-op on non-Windows platforms.
func EnsureRootCAInstalled() error {
	return nil
}

// IsRootCAInstalled is a no-op on non-Windows platforms.
func IsRootCAInstalled() bool {
	return true
}
