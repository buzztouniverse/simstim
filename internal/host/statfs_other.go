//go:build !linux

package host

import "syscall"

// diskFree returns the bytes available to the user on the file system of path.
func diskFree(path string) int64 {
	var st syscall.Statfs_t
	if syscall.Statfs(path, &st) != nil {
		return 0
	}
	return int64(st.Bavail) * int64(st.Bsize)
}
