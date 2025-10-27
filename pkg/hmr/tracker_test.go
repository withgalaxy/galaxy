package hmr

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cameron-webmatter/galaxy/pkg/parser"
)

func TestNewChangeTracker(t *testing.T) {
	tracker := NewChangeTracker()
	if tracker == nil {
		t.Fatal("NewChangeTracker returned nil")
	}
	if tracker.cache == nil {
		t.Error("cache not initialized")
	}
}

func TestChangeTrackerDetectChange_FirstTime(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.gxc")

	content := `---
title: string = "Test"
---
<h1>{title}</h1>
<style>
body { color: red; }
</style>
<script>
console.log("test");
</script>`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tracker := NewChangeTracker()
	diff, err := tracker.DetectChange(testFile)
	if err != nil {
		t.Fatalf("DetectChange failed: %v", err)
	}

	if !diff.TemplateChanged {
		t.Error("expected TemplateChanged on first detection")
	}
	if !diff.StylesChanged {
		t.Error("expected StylesChanged on first detection")
	}
	if !diff.ScriptsChanged {
		t.Error("expected ScriptsChanged on first detection")
	}
	if !diff.FrontmatterChanged {
		t.Error("expected FrontmatterChanged on first detection")
	}
}

func TestChangeTrackerDetectChange_NoChange(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.gxc")

	content := `---
title: string = "Test"
---
<h1>{title}</h1>
<style>
body { color: red; }
</style>`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tracker := NewChangeTracker()

	_, err := tracker.DetectChange(testFile)
	if err != nil {
		t.Fatalf("First DetectChange failed: %v", err)
	}

	diff, err := tracker.DetectChange(testFile)
	if err != nil {
		t.Fatalf("Second DetectChange failed: %v", err)
	}

	if diff.TemplateChanged {
		t.Error("TemplateChanged should be false when no change")
	}
	if diff.StylesChanged {
		t.Error("StylesChanged should be false when no change")
	}
	if diff.ScriptsChanged {
		t.Error("ScriptsChanged should be false when no change")
	}
	if diff.FrontmatterChanged {
		t.Error("FrontmatterChanged should be false when no change")
	}
}

func TestChangeTrackerDetectChange_TemplateOnly(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.gxc")

	content1 := `<h1>Hello</h1>`
	if err := os.WriteFile(testFile, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tracker := NewChangeTracker()
	_, err := tracker.DetectChange(testFile)
	if err != nil {
		t.Fatalf("First DetectChange failed: %v", err)
	}

	content2 := `<h1>Hello World</h1>`
	if err := os.WriteFile(testFile, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	diff, err := tracker.DetectChange(testFile)
	if err != nil {
		t.Fatalf("Second DetectChange failed: %v", err)
	}

	if !diff.TemplateChanged {
		t.Error("expected TemplateChanged")
	}
	if diff.StylesChanged {
		t.Error("StylesChanged should be false")
	}
	if diff.ScriptsChanged {
		t.Error("ScriptsChanged should be false")
	}
	if diff.FrontmatterChanged {
		t.Error("FrontmatterChanged should be false")
	}
}

func TestChangeTrackerDetectChange_StyleOnly(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.gxc")

	content1 := `<h1>Hello</h1>
<style>
body { color: red; }
</style>`
	if err := os.WriteFile(testFile, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tracker := NewChangeTracker()
	_, err := tracker.DetectChange(testFile)
	if err != nil {
		t.Fatalf("First DetectChange failed: %v", err)
	}

	content2 := `<h1>Hello</h1>
<style>
body { color: blue; }
</style>`
	if err := os.WriteFile(testFile, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	diff, err := tracker.DetectChange(testFile)
	if err != nil {
		t.Fatalf("Second DetectChange failed: %v", err)
	}

	if diff.TemplateChanged {
		t.Error("TemplateChanged should be false")
	}
	if !diff.StylesChanged {
		t.Error("expected StylesChanged")
	}
	if diff.ScriptsChanged {
		t.Error("ScriptsChanged should be false")
	}
	if diff.FrontmatterChanged {
		t.Error("FrontmatterChanged should be false")
	}
}

func TestChangeTrackerDetectChange_ScriptOnly(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.gxc")

	content1 := `<h1>Hello</h1>
<script>
console.log("v1");
</script>`
	if err := os.WriteFile(testFile, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tracker := NewChangeTracker()
	_, err := tracker.DetectChange(testFile)
	if err != nil {
		t.Fatalf("First DetectChange failed: %v", err)
	}

	content2 := `<h1>Hello</h1>
<script>
console.log("v2");
</script>`
	if err := os.WriteFile(testFile, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	diff, err := tracker.DetectChange(testFile)
	if err != nil {
		t.Fatalf("Second DetectChange failed: %v", err)
	}

	if diff.TemplateChanged {
		t.Error("TemplateChanged should be false")
	}
	if diff.StylesChanged {
		t.Error("StylesChanged should be false")
	}
	if !diff.ScriptsChanged {
		t.Error("expected ScriptsChanged")
	}
	if diff.FrontmatterChanged {
		t.Error("FrontmatterChanged should be false")
	}
}

func TestChangeTrackerDetectChange_FrontmatterOnly(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.gxc")

	content1 := `---
title: string = "V1"
---
<h1>{title}</h1>`
	if err := os.WriteFile(testFile, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tracker := NewChangeTracker()
	_, err := tracker.DetectChange(testFile)
	if err != nil {
		t.Fatalf("First DetectChange failed: %v", err)
	}

	content2 := `---
title: string = "V2"
---
<h1>{title}</h1>`
	if err := os.WriteFile(testFile, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	diff, err := tracker.DetectChange(testFile)
	if err != nil {
		t.Fatalf("Second DetectChange failed: %v", err)
	}

	if diff.TemplateChanged {
		t.Error("TemplateChanged should be false")
	}
	if diff.StylesChanged {
		t.Error("StylesChanged should be false")
	}
	if diff.ScriptsChanged {
		t.Error("ScriptsChanged should be false")
	}
	if !diff.FrontmatterChanged {
		t.Error("expected FrontmatterChanged")
	}
}

func TestChangeTrackerClear(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.gxc")

	content := `<h1>Hello</h1>`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tracker := NewChangeTracker()
	_, err := tracker.DetectChange(testFile)
	if err != nil {
		t.Fatalf("DetectChange failed: %v", err)
	}

	if len(tracker.cache) == 0 {
		t.Error("cache should not be empty after DetectChange")
	}

	tracker.Clear()

	if len(tracker.cache) != 0 {
		t.Errorf("cache should be empty after Clear, got %d items", len(tracker.cache))
	}
}

func TestChangeTrackerInvalidFile(t *testing.T) {
	tracker := NewChangeTracker()
	_, err := tracker.DetectChange("/nonexistent/file.gxc")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestChangeTrackerInvalidSyntax(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.gxc")

	invalidContent := `<h1>Unclosed tag`
	if err := os.WriteFile(testFile, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tracker := NewChangeTracker()
	diff, err := tracker.DetectChange(testFile)

	if err != nil {
		return
	}

	if diff == nil {
		t.Error("expected diff even with parse errors")
	}
}

func TestDiffComponents(t *testing.T) {
	old := &parser.Component{
		Frontmatter: "title: string = \"Old\"",
		Template:    "<h1>Old</h1>",
		Styles: []parser.Style{
			{Content: "body { color: red; }"},
		},
		Scripts: []parser.Script{
			{Content: "console.log('old');"},
		},
	}

	new := &parser.Component{
		Frontmatter: "title: string = \"New\"",
		Template:    "<h1>New</h1>",
		Styles: []parser.Style{
			{Content: "body { color: blue; }"},
		},
		Scripts: []parser.Script{
			{Content: "console.log('new');"},
		},
	}

	diff := DiffComponents(old, new)

	if !diff.FrontmatterChanged {
		t.Error("expected FrontmatterChanged")
	}
	if !diff.TemplateChanged {
		t.Error("expected TemplateChanged")
	}
	if !diff.StylesChanged {
		t.Error("expected StylesChanged")
	}
	if !diff.ScriptsChanged {
		t.Error("expected ScriptsChanged")
	}
}

func TestDiffComponents_NoChanges(t *testing.T) {
	comp := &parser.Component{
		Frontmatter: "title: string = \"Test\"",
		Template:    "<h1>Test</h1>",
		Styles: []parser.Style{
			{Content: "body { color: red; }"},
		},
		Scripts: []parser.Script{
			{Content: "console.log('test');"},
		},
	}

	diff := DiffComponents(comp, comp)

	if diff.FrontmatterChanged {
		t.Error("FrontmatterChanged should be false")
	}
	if diff.TemplateChanged {
		t.Error("TemplateChanged should be false")
	}
	if diff.StylesChanged {
		t.Error("StylesChanged should be false")
	}
	if diff.ScriptsChanged {
		t.Error("ScriptsChanged should be false")
	}
}

func TestComponentDiff_NeedsFullReload(t *testing.T) {
	tests := []struct {
		name     string
		diff     ComponentDiff
		expected bool
	}{
		{
			name:     "frontmatter changed",
			diff:     ComponentDiff{FrontmatterChanged: true},
			expected: true,
		},
		{
			name:     "scripts changed",
			diff:     ComponentDiff{ScriptsChanged: true},
			expected: true,
		},
		{
			name:     "only styles changed",
			diff:     ComponentDiff{StylesChanged: true},
			expected: false,
		},
		{
			name:     "only template changed",
			diff:     ComponentDiff{TemplateChanged: true},
			expected: false,
		},
		{
			name:     "no changes",
			diff:     ComponentDiff{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.diff.NeedsFullReload(); got != tt.expected {
				t.Errorf("NeedsFullReload() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestComponentDiff_CanHotSwapStyles(t *testing.T) {
	tests := []struct {
		name     string
		diff     ComponentDiff
		expected bool
	}{
		{
			name:     "only styles changed",
			diff:     ComponentDiff{StylesChanged: true},
			expected: true,
		},
		{
			name: "styles and template changed",
			diff: ComponentDiff{
				StylesChanged:   true,
				TemplateChanged: true,
			},
			expected: false,
		},
		{
			name: "styles and scripts changed",
			diff: ComponentDiff{
				StylesChanged:  true,
				ScriptsChanged: true,
			},
			expected: false,
		},
		{
			name: "styles and frontmatter changed",
			diff: ComponentDiff{
				StylesChanged:      true,
				FrontmatterChanged: true,
			},
			expected: false,
		},
		{
			name:     "no changes",
			diff:     ComponentDiff{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.diff.CanHotSwapStyles(); got != tt.expected {
				t.Errorf("CanHotSwapStyles() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestHashString(t *testing.T) {
	str1 := "test string"
	str2 := "test string"
	str3 := "different string"

	hash1 := hashString(str1)
	hash2 := hashString(str2)
	hash3 := hashString(str3)

	if hash1 != hash2 {
		t.Error("same strings should produce same hash")
	}
	if hash1 == hash3 {
		t.Error("different strings should produce different hash")
	}
	if len(hash1) == 0 {
		t.Error("hash should not be empty")
	}
}
