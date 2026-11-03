//go:build !windows

package runner

// readSavedPath is empty: only Windows keeps a PATH for new processes apart from the login shell's, read once at start.
func readSavedPath() (string, error) {
	return "", nil
}
