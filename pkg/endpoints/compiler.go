package endpoints

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type EndpointCompiler struct {
	CacheDir   string
	BaseDir    string
	ModuleName string
	cache      map[string]*cacheEntry
}

type cacheEntry struct {
	endpoint *LoadedEndpoint
	modTime  time.Time
	soPath   string
}

func NewCompiler(baseDir, cacheDir string) *EndpointCompiler {
	moduleName := detectModuleName(baseDir)
	return &EndpointCompiler{
		CacheDir:   cacheDir,
		BaseDir:    baseDir,
		ModuleName: moduleName,
		cache:      make(map[string]*cacheEntry),
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

func (c *EndpointCompiler) Load(filePath string) (*LoadedEndpoint, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	if entry, ok := c.cache[filePath]; ok {
		if entry.modTime.Equal(info.ModTime()) {
			return entry.endpoint, nil
		}
	}

	methods := c.detectMethods(filePath)
	if len(methods) == 0 {
		return nil, fmt.Errorf("no HTTP method handlers found in %s", filePath)
	}

	soPath, err := c.compile(filePath, methods)
	if err != nil {
		return nil, err
	}

	endpoint, err := LoadPlugin(soPath)
	if err != nil {
		return nil, err
	}

	c.cache[filePath] = &cacheEntry{
		endpoint: endpoint,
		modTime:  info.ModTime(),
		soPath:   soPath,
	}

	return endpoint, nil
}

func (c *EndpointCompiler) detectMethods(filePath string) []string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	src := string(content)
	methods := []string{}
	httpMethods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "ALL"}

	for _, method := range httpMethods {
		pattern := fmt.Sprintf("func %s(", method)
		if strings.Contains(src, pattern) {
			methods = append(methods, method)
		}
	}

	return methods
}

func (c *EndpointCompiler) compile(filePath string, methods []string) (string, error) {
	if err := os.MkdirAll(c.CacheDir, 0755); err != nil {
		return "", err
	}

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(filePath)))[:8]
	soPath := filepath.Join(c.CacheDir, fmt.Sprintf("endpoint-%s.so", hash))

	dir := filepath.Dir(filePath)
	relPath, _ := filepath.Rel(c.BaseDir, dir)

	// Sanitize path: replace brackets for valid Go paths
	sanitizedRelPath := strings.ReplaceAll(relPath, "[", "_")
	sanitizedRelPath = strings.ReplaceAll(sanitizedRelPath, "]", "_")

	// Copy source to cache with sanitized path
	cacheSourceDir := filepath.Join(c.CacheDir, "src", sanitizedRelPath)
	if err := os.MkdirAll(cacheSourceDir, 0755); err != nil {
		return "", err
	}

	sourceContent, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// Strip build tags from copied source
	content := string(sourceContent)
	lines := strings.Split(content, "\n")
	var filtered []string
	for _, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "//go:build") && !strings.HasPrefix(strings.TrimSpace(line), "// +build") {
			filtered = append(filtered, line)
		}
	}
	content = strings.Join(filtered, "\n")

	cacheSourcePath := filepath.Join(cacheSourceDir, filepath.Base(filePath))
	if err := os.WriteFile(cacheSourcePath, []byte(content), 0644); err != nil {
		return "", err
	}

	pkgName := filepath.Base(cacheSourceDir)
	importPath := filepath.Join(c.ModuleName, ".galaxy/endpoints/src", sanitizedRelPath)

	var methodExports strings.Builder
	for _, method := range methods {
		methodExports.WriteString(fmt.Sprintf("func %s(ctx *endpoints.Context) error { return %s.%s(ctx) }\n", method, pkgName, method))
	}

	pluginSrc := fmt.Sprintf(`package main

import (
	"github.com/withgalaxy/galaxy/pkg/endpoints"
	%s "%s"
)

%s
`, pkgName, importPath, methodExports.String())

	pluginPath := filepath.Join(c.CacheDir, fmt.Sprintf("endpoint-%s.go", hash))
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

func (c *EndpointCompiler) getPackageName(filePath string) string {
	dir := filepath.Dir(filePath)
	base := filepath.Base(dir)
	if base == "api" || base == "." {
		return "api"
	}
	return base
}
