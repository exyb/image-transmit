//go:build linux

package embedded

import (
	"os"
	"syscall"
)

func execTool(binPath string, args []string) error {
	return syscall.Exec(binPath, args, os.Environ())
}
