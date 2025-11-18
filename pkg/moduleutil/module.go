package moduleutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FindGalaxyModuleRoot attempts to find the local Galaxy module root.
// Returns empty string if not found (indicating published version should be used).
func FindGalaxyModuleRoot() string {
	// Try from executable path (walking up to find go.mod)
	exePath, err := os.Executable()
	if err == nil {
		exePath, _ = filepath.EvalSymlinks(exePath)
		dir := filepath.Dir(exePath)
		for {
			goModPath := filepath.Join(dir, "go.mod")
			if data, err := os.ReadFile(goModPath); err == nil {
				if strings.Contains(string(data), "module github.com/withgalaxy/galaxy") {
					return dir
				}
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// Try using go list to find the module
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/withgalaxy/galaxy")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		path := strings.TrimSpace(string(output))
		// Verify this is actually a local path and not from module cache
		if !strings.Contains(path, "go/pkg/mod") {
			return path
		}
	}

	return ""
}

// GetGalaxyModuleRequirement returns the go.mod require directive for Galaxy.
// If localPath is not empty, it includes a replace directive for development.
// Otherwise, it uses the published version.
func GetGalaxyModuleRequirement(localPath string, version string) string {
	if localPath != "" {
		return "require github.com/withgalaxy/galaxy v0.0.0\n\nreplace github.com/withgalaxy/galaxy => " + localPath + "\n"
	}

	if version[0] != 'v' {
		version = "v" + version
	}
	return "require github.com/withgalaxy/galaxy " + version + "\n"
}
