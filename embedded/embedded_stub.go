//go:build !embedded_tools

package embedded

import "fmt"

func ExtractFromFS(binDir string) error {
	return fmt.Errorf("embedded tools not available in this build (rebuild with -tags embedded_tools)")
}
