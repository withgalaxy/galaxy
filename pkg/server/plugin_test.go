package server

import (
	"testing"
)

func TestNewPluginCompiler(t *testing.T) {
	tmpDir := t.TempDir()

	pc := NewPluginCompiler(".galaxy", "dev-server", "/path/to/galaxy", tmpDir)
	if pc == nil {
		t.Fatal("NewPluginCompiler returned nil")
	}
	if pc.CacheDir != ".galaxy" {
		t.Errorf("expected CacheDir .galaxy, got %s", pc.CacheDir)
	}
	if pc.ModuleName != "dev-server" {
		t.Errorf("expected ModuleName dev-server, got %s", pc.ModuleName)
	}
}

func TestPluginCompiler_CacheDir(t *testing.T) {
	tmpDir := t.TempDir()
	pc := NewPluginCompiler(".galaxy", "dev-server", "/path/to/galaxy", tmpDir)

	if pc.CacheDir != ".galaxy" {
		t.Errorf("expected CacheDir .galaxy, got %s", pc.CacheDir)
	}
}

func TestPluginCompiler_GalaxyPath(t *testing.T) {
	tmpDir := t.TempDir()
	galaxyPath := "/custom/galaxy/path"
	pc := NewPluginCompiler(".galaxy", "dev-server", galaxyPath, tmpDir)

	if pc.GalaxyPath != galaxyPath {
		t.Errorf("expected GalaxyPath %s, got %s", galaxyPath, pc.GalaxyPath)
	}
}

func TestPluginCompiler_ProjectRoot(t *testing.T) {
	tmpDir := t.TempDir()
	pc := NewPluginCompiler(".galaxy", "dev-server", "/path/to/galaxy", tmpDir)

	if pc.ProjectRoot != tmpDir {
		t.Errorf("expected ProjectRoot %s, got %s", tmpDir, pc.ProjectRoot)
	}
}

func TestSanitizeRouteName(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"/"},
		{"/about"},
		{"/blog/posts"},
		{"/api/users/[id]"},
	}

	for _, tt := range tests {
		result := sanitizeRouteName(tt.input)
		if result == "" {
			t.Errorf("sanitizeRouteName(%s) returned empty string", tt.input)
		}
	}
}

func contains(s, substr string) bool {
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
