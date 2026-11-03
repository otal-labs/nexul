//go:build windows

package runner

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows/registry"
)

// readSavedPath is the PATH Windows gives a process started now: the machine's value, then the person's, each expanded.
func readSavedPath() (string, error) {
	machine, err := registryPath(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`)
	if err != nil {
		return "", err
	}
	user, err := registryPath(registry.CURRENT_USER, `Environment`)
	if err != nil {
		return "", err
	}
	return mergePaths(machine, user), nil
}

// registryPath is the Path value under root's key, expanded; a key with none has an empty one.
func registryPath(root registry.Key, key string) (string, error) {
	k, err := registry.OpenKey(root, key, registry.QUERY_VALUE)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", key, err)
	}
	defer func() { _ = k.Close() }() // a key opened only to read has nothing to lose on close
	value, _, err := k.GetStringValue("Path")
	if errors.Is(err, registry.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read Path under %s: %w", key, err)
	}
	expanded, err := registry.ExpandString(value)
	if err != nil {
		return "", fmt.Errorf("expand Path under %s: %w", key, err)
	}
	return expanded, nil
}
