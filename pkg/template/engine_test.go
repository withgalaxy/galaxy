package template

import (
	"strings"
	"testing"

	"github.com/withgalaxy/galaxy/pkg/executor"
)

func TestRenderExpressions(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Set("title", "Hello World")
	ctx.Set("count", int64(42))

	engine := NewEngine(ctx)

	template := `<h1>{title}</h1><p>Count: {count}</p>`
	result, err := engine.Render(template, nil)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	expected := `<h1>Hello World</h1><p>Count: 42</p>`
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestRenderSlots(t *testing.T) {
	ctx := executor.NewContext()
	engine := NewEngine(ctx)

	template := `<div><slot /></div>`
	opts := &RenderOptions{
		Slots: map[string]string{
			"default": "<p>Slot content</p>",
		},
	}

	result, err := engine.Render(template, opts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "Slot content") {
		t.Errorf("Expected slot content in result, got %q", result)
	}
}

func TestRenderNamedSlots(t *testing.T) {
	ctx := executor.NewContext()
	engine := NewEngine(ctx)

	template := `<div><slot name="header" /><slot name="footer" /></div>`
	opts := &RenderOptions{
		Slots: map[string]string{
			"header": "<header>Header</header>",
			"footer": "<footer>Footer</footer>",
		},
	}

	result, err := engine.Render(template, opts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "Header") || !strings.Contains(result, "Footer") {
		t.Errorf("Expected both slots in result, got %q", result)
	}
}

func TestRenderIfDirective(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Set("show", true)
	ctx.Set("hide", false)

	engine := NewEngine(ctx)

	template := `<div galaxy:if={show}>Visible</div><div galaxy:if={hide}>Hidden</div>`
	result, err := engine.Render(template, nil)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "Visible") {
		t.Error("Expected visible div")
	}

	if strings.Contains(result, "galaxy:if") {
		t.Errorf("Result should not contain galaxy:if, got: %s", result)
	}
}

func TestRenderForDirective(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Set("items", []interface{}{"Apple", "Banana", "Cherry"})

	engine := NewEngine(ctx)

	template := `<ul><li galaxy:for={item in items}>{item}</li></ul>`
	result, err := engine.Render(template, nil)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "Apple") {
		t.Error("Expected Apple in result")
	}
	if !strings.Contains(result, "Banana") {
		t.Error("Expected Banana in result")
	}
	if !strings.Contains(result, "Cherry") {
		t.Error("Expected Cherry in result")
	}
}

func TestRenderWithProps(t *testing.T) {
	ctx := executor.NewContext()
	engine := NewEngine(ctx)

	template := `<h1>{title}</h1><p>{description}</p>`
	opts := &RenderOptions{
		Props: map[string]interface{}{
			"title":       "My Title",
			"description": "My Description",
		},
	}

	result, err := engine.Render(template, opts)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "My Title") {
		t.Error("Expected title in result")
	}
	if !strings.Contains(result, "My Description") {
		t.Error("Expected description in result")
	}
}
