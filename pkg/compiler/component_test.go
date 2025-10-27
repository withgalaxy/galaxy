package compiler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cameron-webmatter/galaxy/pkg/executor"
	"github.com/cameron-webmatter/galaxy/pkg/parser"
)

func TestNewComponentCompiler(t *testing.T) {
	cc := NewComponentCompiler("/test/dir")
	if cc == nil {
		t.Fatal("NewComponentCompiler returned nil")
	}
	if cc.BaseDir != "/test/dir" {
		t.Errorf("expected BaseDir /test/dir, got %s", cc.BaseDir)
	}
	if cc.Cache == nil {
		t.Error("Cache not initialized")
	}
	if cc.Resolver == nil {
		t.Error("Resolver not initialized")
	}
}

func TestComponentCompiler_ClearCache(t *testing.T) {
	cc := NewComponentCompiler("/test/dir")

	cc.Cache["test"] = nil
	cc.CollectedStyles = make([]parser.Style, 3)
	cc.UsedComponents = []string{"comp1"}
	cc.componentsSeen = map[string]bool{"comp1": true}

	cc.ClearCache()

	if len(cc.Cache) != 0 {
		t.Error("Cache should be empty after clear")
	}
	if cc.CollectedStyles != nil {
		t.Error("CollectedStyles should be nil after clear")
	}
	if cc.UsedComponents != nil {
		t.Error("UsedComponents should be nil after clear")
	}
	if cc.componentsSeen != nil {
		t.Error("componentsSeen should be nil after clear")
	}
}

func TestComponentCompiler_ResetComponentTracking(t *testing.T) {
	cc := NewComponentCompiler("/test/dir")

	cc.UsedComponents = []string{"comp1", "comp2"}
	cc.componentsSeen = map[string]bool{"comp1": true}

	cc.ResetComponentTracking()

	if cc.UsedComponents != nil {
		t.Error("UsedComponents should be nil after reset")
	}
	if cc.componentsSeen == nil {
		t.Error("componentsSeen should be initialized after reset")
	}
	if len(cc.componentsSeen) != 0 {
		t.Error("componentsSeen should be empty after reset")
	}
}

func TestComponentCompiler_TrackComponent(t *testing.T) {
	cc := NewComponentCompiler("/test/dir")

	cc.trackComponent("/components/Button.gxc")
	cc.trackComponent("/components/Card.gxc")
	cc.trackComponent("/components/Button.gxc")

	if len(cc.UsedComponents) != 2 {
		t.Errorf("expected 2 used components, got %d", len(cc.UsedComponents))
	}

	if !cc.componentsSeen["/components/Button.gxc"] {
		t.Error("Button.gxc should be marked as seen")
	}
	if !cc.componentsSeen["/components/Card.gxc"] {
		t.Error("Card.gxc should be marked as seen")
	}
}

func TestComponentCompiler_ParseAttributes_CurlyBraces(t *testing.T) {
	cc := NewComponentCompiler("/test/dir")
	ctx := executor.NewContext()
	ctx.Set("title", "Hello World")
	ctx.Set("count", 42)

	attrs := `title={title} count={count}`
	props := cc.parseAttributes(attrs, ctx)

	if props["title"] != "Hello World" {
		t.Errorf("expected title 'Hello World', got %v", props["title"])
	}
	if props["count"] != 42 {
		t.Errorf("expected count 42, got %v", props["count"])
	}
}

func TestComponentCompiler_ParseAttributes_DoubleQuotes(t *testing.T) {
	cc := NewComponentCompiler("/test/dir")
	ctx := executor.NewContext()

	attrs := `title="Static Title" class="btn btn-primary"`
	props := cc.parseAttributes(attrs, ctx)

	if props["title"] != "Static Title" {
		t.Errorf("expected title 'Static Title', got %v", props["title"])
	}
	if props["class"] != "btn btn-primary" {
		t.Errorf("expected class 'btn btn-primary', got %v", props["class"])
	}
}

func TestComponentCompiler_ParseAttributes_SingleQuotes(t *testing.T) {
	cc := NewComponentCompiler("/test/dir")
	ctx := executor.NewContext()

	attrs := `title='Single Quoted' type='button'`
	props := cc.parseAttributes(attrs, ctx)

	if props["title"] != "Single Quoted" {
		t.Errorf("expected title 'Single Quoted', got %v", props["title"])
	}
	if props["type"] != "button" {
		t.Errorf("expected type 'button', got %v", props["type"])
	}
}

func TestComponentCompiler_ParseAttributes_Mixed(t *testing.T) {
	cc := NewComponentCompiler("/test/dir")
	ctx := executor.NewContext()
	ctx.Set("dynamic", "Dynamic Value")

	attrs := `title={dynamic} class="static" type='button'`
	props := cc.parseAttributes(attrs, ctx)

	if len(props) != 3 {
		t.Errorf("expected 3 props, got %d", len(props))
	}

	if props["title"] != "Dynamic Value" {
		t.Errorf("expected title 'Dynamic Value', got %v", props["title"])
	}
	if props["class"] != "static" {
		t.Errorf("expected class 'static', got %v", props["class"])
	}
	if props["type"] != "button" {
		t.Errorf("expected type 'button', got %v", props["type"])
	}
}

func TestComponentCompiler_ParseAttributes_UndefinedVariable(t *testing.T) {
	cc := NewComponentCompiler("/test/dir")
	ctx := executor.NewContext()

	attrs := `title={undefined}`
	props := cc.parseAttributes(attrs, ctx)

	if props["title"] != "{undefined}" {
		t.Errorf("expected title '{undefined}', got %v", props["title"])
	}
}

func TestComponentCompiler_LoadComponent_Cache(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.gxc")

	content := `<h1>Test</h1>`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cc := NewComponentCompiler(tmpDir)

	comp1, err := cc.loadComponent(testFile)
	if err != nil {
		t.Fatalf("First loadComponent failed: %v", err)
	}

	comp2, err := cc.loadComponent(testFile)
	if err != nil {
		t.Fatalf("Second loadComponent failed: %v", err)
	}

	if comp1 != comp2 {
		t.Error("loadComponent should return cached component")
	}
}

func TestComponentCompiler_LoadComponent_NotFound(t *testing.T) {
	cc := NewComponentCompiler("/nonexistent")

	_, err := cc.loadComponent("/nonexistent/file.gxc")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestComponentCompiler_ProcessComponentTags_SelfClosing(t *testing.T) {
	tmpDir := t.TempDir()

	compFile := filepath.Join(tmpDir, "components", "Icon.gxc")
	if err := os.MkdirAll(filepath.Dir(compFile), 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	if err := os.WriteFile(compFile, []byte(`<svg>icon</svg>`), 0644); err != nil {
		t.Fatalf("Failed to write component: %v", err)
	}

	cc := NewComponentCompiler(tmpDir)
	cc.Resolver = NewComponentResolver(tmpDir, []string{"components"})
	cc.Resolver.buildComponentIndex()

	template := `<div><Icon /></div>`
	ctx := executor.NewContext()

	result := cc.ProcessComponentTags(template, ctx)

	if !contains(result, "<svg>icon</svg>") {
		t.Errorf("expected icon SVG in result, got: %s", result)
	}
}

func TestComponentCompiler_ProcessComponentTags_WithContent(t *testing.T) {
	tmpDir := t.TempDir()

	cardFile := filepath.Join(tmpDir, "components", "Card.gxc")
	if err := os.MkdirAll(filepath.Dir(cardFile), 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	cardContent := `<div class="card">{#slot default}Card Content{/slot}</div>`
	if err := os.WriteFile(cardFile, []byte(cardContent), 0644); err != nil {
		t.Fatalf("Failed to write component: %v", err)
	}

	cc := NewComponentCompiler(tmpDir)
	cc.Resolver = NewComponentResolver(tmpDir, []string{"components"})
	cc.Resolver.buildComponentIndex()

	template := `<Card>Hello World</Card>`
	ctx := executor.NewContext()

	result := cc.ProcessComponentTags(template, ctx)

	if !contains(result, "Hello World") {
		t.Errorf("expected slot content in result, got: %s", result)
	}
}

func TestComponentCompiler_ProcessComponentTags_ComponentNotFound(t *testing.T) {
	cc := NewComponentCompiler("/test")
	cc.Resolver = NewComponentResolver("/test", []string{"components"})

	template := `<NonExistent />`
	ctx := executor.NewContext()

	result := cc.ProcessComponentTags(template, ctx)

	if !contains(result, "Component resolution error") {
		t.Errorf("expected error comment in result, got: %s", result)
	}
}

func TestComponentCompiler_ProcessComponentTags_MismatchedTags(t *testing.T) {
	cc := NewComponentCompiler("/test")

	template := `<Card>content</OtherTag>`
	ctx := executor.NewContext()

	result := cc.ProcessComponentTags(template, ctx)

	if !contains(result, "content") {
		t.Errorf("should preserve content with mismatched tags, got: %s", result)
	}
}

func TestComponentCompiler_SetResolver(t *testing.T) {
	cc := NewComponentCompiler("/test")
	originalResolver := cc.Resolver

	newResolver := NewComponentResolver("/other", nil)
	cc.SetResolver(newResolver)

	if cc.Resolver == originalResolver {
		t.Error("Resolver should have changed")
	}
	if cc.Resolver != newResolver {
		t.Error("Resolver should be the new resolver")
	}
}

func TestComponentCompiler_Compile_Simple(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.gxc")

	content := `<h1>{title}</h1>`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cc := NewComponentCompiler(tmpDir)
	props := map[string]interface{}{"title": "Hello"}

	result, err := cc.Compile(testFile, props, nil)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	if !contains(result, "Hello") {
		t.Errorf("expected 'Hello' in result, got: %s", result)
	}
}

func TestComponentCompiler_Compile_WithFrontmatter(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.gxc")

	content := `---
computed := title + " World"
---
<h1>{computed}</h1>`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cc := NewComponentCompiler(tmpDir)
	props := map[string]interface{}{"title": "Hello"}

	result, err := cc.Compile(testFile, props, nil)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	if !contains(result, "Hello World") {
		t.Errorf("expected 'Hello World' in result, got: %s", result)
	}
}

func TestComponentCompiler_Compile_CollectsStyles(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.gxc")

	content := `<h1>Test</h1>
<style>
body { color: red; }
</style>`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cc := NewComponentCompiler(tmpDir)

	_, err := cc.Compile(testFile, nil, nil)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	if len(cc.CollectedStyles) == 0 {
		t.Error("expected styles to be collected")
	}
}

func TestComponentCompiler_CompileWithContext_InheritsContext(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.gxc")

	content := `<h1>{parentVar}</h1>`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cc := NewComponentCompiler(tmpDir)

	parentCtx := executor.NewContext()
	parentCtx.Set("parentVar", "From Parent")

	result, err := cc.CompileWithContext(testFile, nil, nil, parentCtx)
	if err != nil {
		t.Fatalf("CompileWithContext failed: %v", err)
	}

	if !contains(result, "From Parent") {
		t.Errorf("expected 'From Parent' in result, got: %s", result)
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
