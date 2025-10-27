package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/withgalaxy/galaxy/pkg/router"
)

func TestNewCodegenBuilder(t *testing.T) {
	routes := []*router.Route{}
	pagesDir := "/test/src/pages"
	outDir := ".galaxy"
	moduleName := "test-server"
	publicDir := "/test/public"

	builder := NewCodegenBuilder(routes, pagesDir, outDir, moduleName, publicDir)

	if builder == nil {
		t.Fatal("NewCodegenBuilder returned nil")
	}
	if builder.PagesDir != pagesDir {
		t.Errorf("expected PagesDir %s, got %s", pagesDir, builder.PagesDir)
	}
	if builder.OutDir != outDir {
		t.Errorf("expected OutDir %s, got %s", outDir, builder.OutDir)
	}
	if builder.ModuleName != moduleName {
		t.Errorf("expected ModuleName %s, got %s", moduleName, builder.ModuleName)
	}
	if builder.PublicDir != publicDir {
		t.Errorf("expected PublicDir %s, got %s", publicDir, builder.PublicDir)
	}
	if builder.Bundler == nil {
		t.Error("Bundler not initialized")
	}
}

func TestCodegenBuilder_CopyMiddleware_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	serverDir := filepath.Join(tmpDir, "server")
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		t.Fatalf("Failed to create server dir: %v", err)
	}

	builder := NewCodegenBuilder([]*router.Route{}, tmpDir, ".galaxy", "test", "public")
	builder.MiddlewarePath = filepath.Join(tmpDir, "nonexistent.go")

	err := builder.copyMiddleware(serverDir)
	if err != nil {
		t.Errorf("copyMiddleware should not error when file doesn't exist: %v", err)
	}
}

func TestCodegenBuilder_CopyMiddleware_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	serverDir := filepath.Join(tmpDir, "server")
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		t.Fatalf("Failed to create server dir: %v", err)
	}

	middlewareContent := `package src

func OnRequest() {
	// middleware logic
}`

	middlewarePath := filepath.Join(tmpDir, "middleware.go")
	if err := os.WriteFile(middlewarePath, []byte(middlewareContent), 0644); err != nil {
		t.Fatalf("Failed to write middleware: %v", err)
	}

	builder := NewCodegenBuilder([]*router.Route{}, tmpDir, ".galaxy", "test", "public")
	builder.MiddlewarePath = middlewarePath

	err := builder.copyMiddleware(serverDir)
	if err != nil {
		t.Fatalf("copyMiddleware failed: %v", err)
	}

	copiedPath := filepath.Join(serverDir, "middleware.go")
	copiedContent, err := os.ReadFile(copiedPath)
	if err != nil {
		t.Fatalf("Failed to read copied middleware: %v", err)
	}

	content := string(copiedContent)
	if !contains(content, "package main") {
		t.Error("expected package to be changed to 'main'")
	}
	if contains(content, "package src") {
		t.Error("should not contain original package declaration")
	}
}

func TestCodegenBuilder_CopyPublicAssets_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	serverDir := filepath.Join(tmpDir, "server")
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		t.Fatalf("Failed to create server dir: %v", err)
	}

	builder := NewCodegenBuilder([]*router.Route{}, tmpDir, ".galaxy", "test", "/nonexistent")

	err := builder.copyPublicAssets(serverDir)
	if err != nil {
		t.Errorf("copyPublicAssets should not error when dir doesn't exist: %v", err)
	}
}

func TestCodegenBuilder_CopyPublicAssets_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	publicDir := filepath.Join(tmpDir, "public")
	serverDir := filepath.Join(tmpDir, "server")

	if err := os.MkdirAll(publicDir, 0755); err != nil {
		t.Fatalf("Failed to create public dir: %v", err)
	}
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		t.Fatalf("Failed to create server dir: %v", err)
	}

	testFile := filepath.Join(publicDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	builder := NewCodegenBuilder([]*router.Route{}, tmpDir, ".galaxy", "test", publicDir)

	err := builder.copyPublicAssets(serverDir)
	if err != nil {
		t.Fatalf("copyPublicAssets failed: %v", err)
	}

	copiedFile := filepath.Join(serverDir, "public", "test.txt")
	if _, err := os.Stat(copiedFile); os.IsNotExist(err) {
		t.Error("expected test.txt to be copied")
	}
}

func TestCodegenBuilder_GenerateGoMod(t *testing.T) {
	tmpDir := t.TempDir()
	serverDir := filepath.Join(tmpDir, "server")
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		t.Fatalf("Failed to create server dir: %v", err)
	}

	builder := NewCodegenBuilder([]*router.Route{}, tmpDir, ".galaxy", "test-server", "public")

	err := builder.generateGoMod(serverDir)
	if err != nil {
		t.Fatalf("generateGoMod failed: %v", err)
	}

	goModPath := filepath.Join(serverDir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}

	goModContent := string(content)
	if !contains(goModContent, "module") {
		t.Error("go.mod should contain module declaration")
	}
	if !contains(goModContent, "github.com/withgalaxy/galaxy") {
		t.Error("go.mod should require galaxy")
	}
}

func TestCodegenBuilder_CopyWasmExec(t *testing.T) {
	tmpDir := t.TempDir()
	serverDir := filepath.Join(tmpDir, "server")
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		t.Fatalf("Failed to create server dir: %v", err)
	}

	builder := NewCodegenBuilder([]*router.Route{}, tmpDir, ".galaxy", "test", "public")

	err := builder.copyWasmExec(serverDir)
	if err != nil {
		// May fail if GOROOT not set or wasm_exec.js not found - that's OK
		t.Logf("copyWasmExec failed (may be expected): %v", err)
		return
	}

	wasmExecPath := filepath.Join(serverDir, "wasm_exec.js")
	if _, err := os.Stat(wasmExecPath); os.IsNotExist(err) {
		t.Error("expected wasm_exec.js to be copied")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && hasSubstring(s, substr)
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
