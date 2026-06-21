package embedded

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ToolDef struct {
	Name        string
	Category    string
	Description string
	Symlink     bool
}

var ToolRegistry = []ToolDef{
	{Name: "skopeo", Category: "container", Description: "Container image registry tool", Symlink: true},
	{Name: "ctr", Category: "container", Description: "containerd CLI", Symlink: true},
	{Name: "crictl", Category: "container", Description: "CRI-compatible runtime CLI", Symlink: true},
	{Name: "nerdctl", Category: "container", Description: "Docker-compatible CLI for containerd", Symlink: true},
	{Name: "regctl", Category: "container", Description: "Container registry client (regclient)", Symlink: true},
	{Name: "mc", Category: "storage", Description: "MinIO/S3 compatible object storage client", Symlink: true},
	{Name: "redis-cli", Category: "database", Description: "Redis command-line client", Symlink: true},
}

var KnownTools []string
var KnownToolMap map[string]ToolDef

func init() {
	KnownToolMap = make(map[string]ToolDef)
	for _, t := range ToolRegistry {
		KnownTools = append(KnownTools, t.Name)
		KnownToolMap[t.Name] = t
	}
}

func HandleMulticall() bool {
	progName := filepath.Base(os.Args[0])

	if tool, ok := KnownToolMap[progName]; ok {
		if tool.Symlink {
			if err := RunTool(tool.Name, os.Args[1:]); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
			return true
		}
	}

	if len(os.Args) > 1 {
		arg := os.Args[1]
		if !strings.HasPrefix(arg, "-") {
			if _, ok := KnownToolMap[arg]; ok {
				if err := RunTool(arg, os.Args[2:]); err != nil {
					fmt.Fprintf(os.Stderr, "%v\n", err)
					os.Exit(1)
				}
				return true
			}
		}
	}

	return false
}
