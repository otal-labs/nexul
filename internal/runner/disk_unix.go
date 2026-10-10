//go:build !windows

package runner

import "golang.org/x/sys/unix"

// freeDisk is the space left for the person in the file system holding path, in bytes.
func freeDisk(path string) (int64, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, err
	}
	return int64(st.Bavail) * int64(st.Bsize), nil
}
