//go:build windows

package runner

import "golang.org/x/sys/windows"

// freeDisk is the space left for the person on the volume holding path, in bytes.
func freeDisk(path string) (int64, error) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var free uint64
	if err := windows.GetDiskFreeSpaceEx(name, &free, nil, nil); err != nil {
		return 0, err
	}
	return int64(free), nil
}
