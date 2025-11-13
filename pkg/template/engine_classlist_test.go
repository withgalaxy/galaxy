package template

import (
	"strings"
	"testing"

	"github.com/withgalaxy/galaxy/pkg/executor"
)

func TestClassListBasic(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Set("isActive", true)
	ctx.Set("isDisabled", false)

	template := `<div classList={{"active": isActive, "disabled": isDisabled}}>Test</div>`
	engine := NewEngine(ctx)
	result, err := engine.Render(template, nil)

	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	if !strings.Contains(result, `class="active"`) {
		t.Errorf("Expected active class, got: %s", result)
	}
	if strings.Contains(result, "disabled") {
		t.Errorf("Should not have disabled class, got: %s", result)
	}
}

func TestClassListMultiple(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Set("primary", true)
	ctx.Set("large", true)
	ctx.Set("disabled", false)

	template := `<div classList={{"btn-primary": primary, "btn-large": large, "btn-disabled": disabled}}>Button</div>`
	engine := NewEngine(ctx)
	result, err := engine.Render(template, nil)

	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	if !strings.Contains(result, "btn-primary") {
		t.Error("Expected btn-primary class")
	}
	if !strings.Contains(result, "btn-large") {
		t.Error("Expected btn-large class")
	}
	if strings.Contains(result, "btn-disabled") {
		t.Error("Should not have btn-disabled class")
	}
}

func TestClassListMergeWithStatic(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Set("isActive", true)

	template := `<div class="base container" classList={{"active": isActive}}>Test</div>`
	engine := NewEngine(ctx)
	result, err := engine.Render(template, nil)

	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	if !strings.Contains(result, "base") {
		t.Error("Expected base class")
	}
	if !strings.Contains(result, "container") {
		t.Error("Expected container class")
	}
	if !strings.Contains(result, "active") {
		t.Error("Expected active class")
	}
}

func TestClassListComparison(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Set("count", int64(10))
	ctx.Set("status", "error")

	template := `<div classList={{"many": count > 5, "error": status == "error"}}>Test</div>`
	engine := NewEngine(ctx)
	result, err := engine.Render(template, nil)

	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	if !strings.Contains(result, "many") {
		t.Error("Expected many class (count > 5)")
	}
	if !strings.Contains(result, "error") {
		t.Error("Expected error class")
	}
}

func TestClassListEmpty(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Set("isActive", false)
	ctx.Set("isDisabled", false)

	template := `<div classList={{"active": isActive, "disabled": isDisabled}}>Test</div>`
	engine := NewEngine(ctx)
	result, err := engine.Render(template, nil)

	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	if strings.Contains(result, "active") || strings.Contains(result, "disabled") {
		t.Errorf("Should have no classes, got: %s", result)
	}
}

func TestClassListNegation(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Set("visible", false)

	template := `<div classList={{"hidden": !visible}}>Test</div>`
	engine := NewEngine(ctx)
	result, err := engine.Render(template, nil)

	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	if !strings.Contains(result, "hidden") {
		t.Error("Expected hidden class (!visible where visible=false)")
	}
}

func TestClassListDuplicates(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Set("isActive", true)

	template := `<div class="active base" classList={{"active": isActive}}>Test</div>`
	engine := NewEngine(ctx)
	result, err := engine.Render(template, nil)

	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	count := strings.Count(result, "active")
	if count > 1 {
		t.Errorf("Duplicate 'active' class found: %s", result)
	}
}

func TestClassListWithDirectives(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Set("show", true)
	ctx.Set("isActive", true)

	template := `<div galaxy:if={show} classList={{"active": isActive}}>Test</div>`
	engine := NewEngine(ctx)
	result, err := engine.Render(template, nil)

	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	if !strings.Contains(result, "active") {
		t.Error("Expected active class with galaxy:if directive")
	}
}
