//go:build !linux

package embedded

import (
	"os"
	"os/exec"
)

func execTool(binPath string, args []string) error {
	cmd := exec.Command(binPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
