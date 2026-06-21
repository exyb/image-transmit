package embedded

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	extractDir  string
	extractOnce sync.Once
	extractErr  error
)

func GetExtractDir() (string, error) {
	extractOnce.Do(func() {
		ex, err := os.Executable()
		if err != nil {
			extractErr = err
			return
		}
		exPath := filepath.Dir(ex)
		realPath, err := filepath.EvalSymlinks(exPath)
		if err == nil {
			exPath = realPath
		}
		extractDir = filepath.Join(exPath, ".image-transmit-data")
		if err := os.MkdirAll(extractDir, 0755); err != nil {
			extractErr = err
			return
		}
		extractErr = extractIfNeeded(extractDir)
	})
	return extractDir, extractErr
}

func extractIfNeeded(dir string) error {
	binDir := filepath.Join(dir, "bin")
	marker := filepath.Join(dir, ".extracted")

	ex, _ := os.Executable()
	exInfo, _ := os.Stat(ex)
	expectedMarker := ""
	if exInfo != nil {
		expectedMarker = fmt.Sprintf("%d_%d", exInfo.Size(), exInfo.ModTime().UnixNano())
	}

	data, err := os.ReadFile(marker)
	if err == nil && string(data) == expectedMarker {
		return nil
	}

	os.RemoveAll(binDir)
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	if err := ExtractFromFS(binDir); err != nil {
		return err
	}

	if expectedMarker != "" {
		os.WriteFile(marker, []byte(expectedMarker), 0644)
	}

	return nil
}

func RunTool(name string, args []string) error {
	dir, err := GetExtractDir()
	if err != nil {
		return fmt.Errorf("failed to prepare tool %s: %w", name, err)
	}

	binPath := filepath.Join(dir, "bin", name)
	if _, err := os.Stat(binPath); err != nil {
		toolPath, lookupErr := findToolOnPath(name)
		if lookupErr == nil {
			binPath = toolPath
		} else {
			return fmt.Errorf("tool %s not found (not embedded and not on PATH)", name)
		}
	}

	newArgs := append([]string{name}, args...)
	return execTool(binPath, newArgs)
}

func findToolOnPath(name string) (string, error) {
	pathEnv := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(pathEnv) {
		p := filepath.Join(dir, name)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p, nil
		}
	}
	return "", fmt.Errorf("%s not found on PATH", name)
}
