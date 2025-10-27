package template

import (
	"strings"
	"testing"

	"github.com/withgalaxy/galaxy/pkg/executor"
)

func TestElseDirective(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Execute(`var show = 0`)

	engine := NewEngine(ctx)
	template := `<div galaxy:if={show}>Yes</div>
<div galaxy:else>No</div>`

	result, _ := engine.Render(template, nil)

	if !strings.Contains(result, "<div>No</div>") {
		t.Errorf("expected else branch, got: %s", result)
	}
	if strings.Contains(result, "Yes") {
		t.Errorf("if branch should not render, got: %s", result)
	}
}

func TestElsifDirective(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Execute(`var status = "pending"`)
	ctx.Set("status", "pending")

	engine := NewEngine(ctx)
	template := `<div galaxy:if={active}>Active</div>
<div galaxy:elsif={status}>Pending</div>
<div galaxy:else>Other</div>`

	result, _ := engine.Render(template, nil)

	if !strings.Contains(result, "Pending") {
		t.Errorf("expected elsif branch, got: %s", result)
	}
	if strings.Contains(result, "Active") || strings.Contains(result, "Other") {
		t.Errorf("only elsif should render, got: %s", result)
	}
}

func TestIfElsifElseChain(t *testing.T) {
	tests := []struct {
		varName  string
		expected string
	}{
		{"one", "One"},
		{"two", "Two"},
		{"three", "Three"},
		{"other", "Other"},
	}

	for _, test := range tests {
		ctx := executor.NewContext()
		ctx.Set(test.varName, 1)

		engine := NewEngine(ctx)
		template := `<p galaxy:if={one}>One</p>
<p galaxy:elsif={two}>Two</p>
<p galaxy:elsif={three}>Three</p>
<p galaxy:else>Other</p>`

		result, _ := engine.Render(template, nil)

		if !strings.Contains(result, test.expected) {
			t.Errorf("%s: expected %s, got: %s", test.varName, test.expected, result)
		}
	}
}

func TestNestedIfWithElse(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Execute(`var outer = 1`)
	ctx.Execute(`var inner = 0`)

	engine := NewEngine(ctx)
	template := `<div galaxy:if={outer}>
    <span galaxy:if={inner}>Inner Yes</span>
    <span galaxy:else>Inner No</span>
</div>
<div galaxy:else>Outer No</div>`

	result, _ := engine.Render(template, nil)

	if !strings.Contains(result, "Inner No") {
		t.Errorf("expected inner else, got: %s", result)
	}
	if strings.Contains(result, "Outer No") || strings.Contains(result, "Inner Yes") {
		t.Errorf("wrong branches rendered, got: %s", result)
	}
}

func TestMultipleElsifBranches(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Execute(`var c = 1`)

	engine := NewEngine(ctx)
	template := `<p galaxy:if={a}>A</p>
<p galaxy:elsif={b}>B</p>
<p galaxy:elsif={c}>C</p>
<p galaxy:elsif={d}>D</p>
<p galaxy:else>F</p>`

	result, _ := engine.Render(template, nil)

	if !strings.Contains(result, "<p>C</p>") {
		t.Errorf("expected C, got: %s", result)
	}
	if strings.Contains(result, ">A<") || strings.Contains(result, ">B<") || strings.Contains(result, ">D<") || strings.Contains(result, ">F<") {
		t.Errorf("only C should render, got: %s", result)
	}
}

func TestElseWithAttributes(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Execute(`var show = 0`)

	engine := NewEngine(ctx)
	template := `<div galaxy:if={show} class="yes">Yes</div>
<div galaxy:else class="no" data-test="true">No</div>`

	result, _ := engine.Render(template, nil)

	if !strings.Contains(result, `class="no"`) {
		t.Errorf("expected class attribute, got: %s", result)
	}
	if !strings.Contains(result, `data-test="true"`) {
		t.Errorf("expected data-test attribute, got: %s", result)
	}
}

func TestIfTrueSkipsElse(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Execute(`var show = 1`)

	engine := NewEngine(ctx)
	template := `<div galaxy:if={show}>Yes</div>
<div galaxy:else>No</div>`

	result, _ := engine.Render(template, nil)

	if !strings.Contains(result, "<div>Yes</div>") {
		t.Errorf("expected if branch, got: %s", result)
	}
	if strings.Contains(result, "No") {
		t.Errorf("else should not render, got: %s", result)
	}
}

func TestElsifWithAttributes(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Execute(`var pending = 1`)

	engine := NewEngine(ctx)
	template := `<span galaxy:if={active} class="active">Active</span>
<span galaxy:elsif={pending} class="pending">Pending</span>
<span galaxy:else class="other">Other</span>`

	result, _ := engine.Render(template, nil)

	if !strings.Contains(result, `class="pending"`) {
		t.Errorf("expected pending class, got: %s", result)
	}
	if !strings.Contains(result, "Pending") {
		t.Errorf("expected Pending text, got: %s", result)
	}
}
