package middleware

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type MiddlewareCompiler struct {
	CacheDir   string
	BaseDir    string
	ModuleName string
	cache      *cacheEntry
}

type cacheEntry struct {
	middleware *LoadedMiddleware
	modTime    time.Time
	soPath     string
}

func NewCompiler(baseDir, cacheDir string) *MiddlewareCompiler {
	moduleName := detectModuleName(baseDir)
	return &MiddlewareCompiler{
		CacheDir:   cacheDir,
		BaseDir:    baseDir,
		ModuleName: moduleName,
	}
}

func detectModuleName(baseDir string) string {
	modPath := filepath.Join(baseDir, "go.mod")
	content, err := os.ReadFile(modPath)
	if err != nil {
		return filepath.Base(baseDir)
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module"))
		}
	}

	return filepath.Base(baseDir)
}

func (c *MiddlewareCompiler) Load(filePath string) (*LoadedMiddleware, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	if c.cache != nil && c.cache.modTime.Equal(info.ModTime()) {
		return c.cache.middleware, nil
	}

	hasOnRequest, hasSequence := c.detectFunctions(filePath)
	if !hasOnRequest && !hasSequence {
		return nil, fmt.Errorf("no OnRequest or Sequence functions found in %s", filePath)
	}

	soPath, err := c.compile(filePath, hasOnRequest, hasSequence)
	if err != nil {
		return nil, err
	}

	loaded, err := LoadPlugin(soPath)
	if err != nil {
		return nil, err
	}

	c.cache = &cacheEntry{
		middleware: loaded,
		modTime:    info.ModTime(),
		soPath:     soPath,
	}

	return loaded, nil
}

func (c *MiddlewareCompiler) detectFunctions(filePath string) (hasOnRequest, hasSequence bool) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false, false
	}

	src := string(content)
	hasOnRequest = strings.Contains(src, "func OnRequest(")
	hasSequence = strings.Contains(src, "func Sequence()")

	return
}

func (c *MiddlewareCompiler) compile(filePath string, hasOnRequest, hasSequence bool) (string, error) {
	if err := os.MkdirAll(c.CacheDir, 0755); err != nil {
		return "", err
	}

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(filePath)))[:8]
	soPath := filepath.Join(c.CacheDir, fmt.Sprintf("middleware-%s.so", hash))

	srcDir := filepath.Dir(filePath)
	relPath, _ := filepath.Rel(c.BaseDir, srcDir)
	importPath := filepath.Join(c.ModuleName, relPath)

	var exports strings.Builder
	if hasOnRequest {
		exports.WriteString("func OnRequest(ctx *middleware.Context, next func() error) error {\n")
		exports.WriteString("    return usermw.OnRequest(ctx, next)\n")
		exports.WriteString("}\n\n")
	}
	if hasSequence {
		exports.WriteString("func Sequence() []middleware.Middleware {\n")
		exports.WriteString("    return usermw.Sequence()\n")
		exports.WriteString("}\n")
	}

	pluginSrc := fmt.Sprintf(`package main

import (
	"github.com/withgalaxy/galaxy/pkg/middleware"
	usermw "%s"
)

%s
`, importPath, exports.String())

	pluginPath := filepath.Join(c.CacheDir, fmt.Sprintf("middleware-%s.go", hash))
	if err := os.WriteFile(pluginPath, []byte(pluginSrc), 0644); err != nil {
		return "", err
	}

	cmd := exec.Command("go", "build", "-buildmode=plugin", "-o", soPath, pluginPath)
	cmd.Dir = c.BaseDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("compile plugin: %w\n%s", err, string(output))
	}

	return soPath, nil
}
