//go:build embedded_tools

package embedded

import (
	"compress/gzip"
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

//go:embed bin/*.gz
var binFS embed.FS

func ExtractFromFS(binDir string) error {
	entries, err := binFS.ReadDir("bin")
	if err != nil {
		return fmt.Errorf("failed to read embedded bin directory: %w", err)
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".gz") {
			continue
		}

		srcPath := "bin/" + entry.Name()
		dstName := strings.TrimSuffix(entry.Name(), ".gz")
		dstName = decodeName(dstName)
		dstPath := filepath.Join(binDir, dstName)

		if err := extractOne(srcPath, dstPath); err != nil {
			return err
		}
	}

	return nil
}

func extractOne(srcPath, dstPath string) error {
	f, err := binFS.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open embedded file %s: %w", srcPath, err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader for %s: %w", srcPath, err)
	}
	defer gz.Close()

	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", dstPath, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, gz); err != nil {
		return fmt.Errorf("failed to decompress %s: %w", srcPath, err)
	}

	return nil
}

func decodeName(name string) string {
	return strings.ReplaceAll(name, "_", "-")
}

func init() {
	_ = filepath.Join
}
