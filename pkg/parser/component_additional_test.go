package parser

import (
	"testing"
)

func TestParse_EmptyContent(t *testing.T) {
	comp, err := Parse("")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if comp == nil {
		t.Fatal("expected component, got nil")
	}
	if comp.Template != "" {
		t.Error("expected empty template")
	}
}

func TestParse_OnlyTemplate(t *testing.T) {
	content := "<h1>Hello World</h1>"
	comp, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if comp.Template != content {
		t.Errorf("expected template %s, got %s", content, comp.Template)
	}
	if comp.Frontmatter != "" {
		t.Error("expected no frontmatter")
	}
}

func TestParse_FrontmatterOnly(t *testing.T) {
	content := `---
title: string = "Test"
---`
	comp, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if !contains(comp.Frontmatter, "title") {
		t.Error("expected frontmatter to contain title")
	}
	if comp.Template != "" {
		t.Error("expected empty template")
	}
}

func TestParse_MultipleScripts(t *testing.T) {
	content := `<h1>Test</h1>
<script>
console.log("js1");
</script>
<script>
console.log("js2");
</script>`
	comp, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(comp.Scripts) != 2 {
		t.Errorf("expected 2 scripts, got %d", len(comp.Scripts))
	}
}

func TestParse_MultipleStyles(t *testing.T) {
	content := `<h1>Test</h1>
<style>
.a { color: red; }
</style>
<style scoped>
.b { color: blue; }
</style>`
	comp, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(comp.Styles) != 2 {
		t.Errorf("expected 2 styles, got %d", len(comp.Styles))
	}
	if comp.Styles[0].Scoped {
		t.Error("first style should not be scoped")
	}
	if !comp.Styles[1].Scoped {
		t.Error("second style should be scoped")
	}
}

func TestParse_ScopedStyle(t *testing.T) {
	content := `<style scoped>
.test { color: red; }
</style>`
	comp, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(comp.Styles) != 1 {
		t.Fatal("expected 1 style")
	}
	if !comp.Styles[0].Scoped {
		t.Error("expected scoped style")
	}
}

func TestParse_ModuleScript(t *testing.T) {
	content := `<script type="module">
import { test } from "./test.js";
</script>`
	comp, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(comp.Scripts) != 1 {
		t.Fatal("expected 1 script")
	}
	if !comp.Scripts[0].IsModule {
		t.Error("expected module script")
	}
	if comp.Scripts[0].Language != "javascript" {
		t.Errorf("expected javascript, got %s", comp.Scripts[0].Language)
	}
}

func TestParse_GoScript(t *testing.T) {
	content := `<script>
func main() {
	fmt.Println("test")
}
</script>`
	comp, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(comp.Scripts) != 1 {
		t.Fatal("expected 1 script")
	}
	if comp.Scripts[0].Language != "go" {
		t.Errorf("expected go, got %s", comp.Scripts[0].Language)
	}
}

func TestParse_ExpressionInTemplate(t *testing.T) {
	content := `<h1>{title}</h1>`
	comp, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if !contains(comp.Template, "{title}") {
		t.Error("expected template to contain expression")
	}
}

func TestParse_IfDirective(t *testing.T) {
	content := `{#if condition}
<p>Show</p>
{/if}`
	comp, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if !contains(comp.Template, "{#if") {
		t.Error("expected template to contain if directive")
	}
}

func TestParse_EachDirective(t *testing.T) {
	content := `{#each items as item}
<li>{item}</li>
{/each}`
	comp, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if !contains(comp.Template, "{#each") {
		t.Error("expected template to contain each directive")
	}
}

func TestParse_FullComponent(t *testing.T) {
	content := `---
title: string = "Test"
count: int = 0
---
<h1>{title}</h1>
<p>Count: {count}</p>
<style scoped>
h1 { color: blue; }
</style>
<script>
console.log("loaded");
</script>`
	comp, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if comp.Frontmatter == "" {
		t.Error("expected frontmatter")
	}
	if comp.Template == "" {
		t.Error("expected template")
	}
	if len(comp.Styles) == 0 {
		t.Error("expected styles")
	}
	if len(comp.Scripts) == 0 {
		t.Error("expected scripts")
	}
}

func TestParse_NestedExpressions(t *testing.T) {
	content := `<div>{user.name}</div>
<div>{items[0].title}</div>`
	comp, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if !contains(comp.Template, "{user.name}") {
		t.Error("expected user.name expression")
	}
}

func TestParse_WhitespaceHandling(t *testing.T) {
	content := `---
title: string = "Test"
---

<h1>Hello</h1>

<style>
  .test { }
</style>

<script>
  console.log("test");
</script>`
	comp, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if comp.Frontmatter == "" {
		t.Error("expected frontmatter")
	}
	if !contains(comp.Template, "<h1>Hello</h1>") {
		t.Error("expected template to contain h1")
	}
}

func TestParse_MalformedFrontmatter(t *testing.T) {
	content := `---
title: "unclosed
---
<h1>Test</h1>`
	comp, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if comp.Frontmatter == "" {
		t.Error("expected frontmatter even if malformed")
	}
}

func TestParse_ScriptWithAttributes(t *testing.T) {
	content := `<script type="module" defer>
import test from "./test";
</script>`
	comp, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(comp.Scripts) != 1 {
		t.Fatal("expected 1 script")
	}
	if !comp.Scripts[0].IsModule {
		t.Error("expected module script")
	}
}

func TestParse_StyleWithAttributes(t *testing.T) {
	content := `<style scoped type="text/css">
.test { color: red; }
</style>`
	comp, err := Parse(content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(comp.Styles) != 1 {
		t.Fatal("expected 1 style")
	}
	if !comp.Styles[0].Scoped {
		t.Error("expected scoped style")
	}
}

func TestDetectLanguage_JavaScript(t *testing.T) {
	tests := []string{
		"console.log('test')",
		"const x = 10;",
		"import test from './test'",
	}

	for _, content := range tests {
		lang := detectLanguage(content)
		if lang != "javascript" {
			t.Errorf("expected javascript for %s, got %s", content, lang)
		}
	}
}

func TestDetectLanguage_Go(t *testing.T) {
	tests := []string{
		"func main() {}",
		"package main",
		"fmt.Println()",
		"var x int = 10",
	}

	for _, content := range tests {
		lang := detectLanguage(content)
		if lang != "go" {
			t.Errorf("expected go for %s, got %s", content, lang)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && hasSubstr(s, substr)
}

func hasSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
