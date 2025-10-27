package template

import (
	"strings"
	"testing"

	"github.com/withgalaxy/galaxy/pkg/executor"
)

func TestComparisonEqual(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Set("status", "active")

	engine := NewEngine(ctx)
	template := `<div galaxy:if={status == "active"}>Active</div>
<div galaxy:else>Inactive</div>`

	result, _ := engine.Render(template, nil)

	if !strings.Contains(result, "Active") {
		t.Errorf("expected Active, got: %s", result)
	}
	if strings.Contains(result, "Inactive") {
		t.Errorf("should not contain Inactive, got: %s", result)
	}
}

func TestComparisonNotEqual(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Set("status", "pending")

	engine := NewEngine(ctx)
	template := `<div galaxy:if={status != "active"}>Not Active</div>
<div galaxy:else>Active</div>`

	result, _ := engine.Render(template, nil)

	if !strings.Contains(result, "Not Active") {
		t.Errorf("expected Not Active, got: %s", result)
	}
}

func TestComparisonGreaterThan(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Set("score", int64(85))

	engine := NewEngine(ctx)
	template := `<div galaxy:if={score > 80}>High</div>
<div galaxy:else>Low</div>`

	result, _ := engine.Render(template, nil)

	if !strings.Contains(result, "High") {
		t.Errorf("expected High, got: %s", result)
	}
}

func TestComparisonLessThan(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Set("score", int64(75))

	engine := NewEngine(ctx)
	template := `<div galaxy:if={score < 80}>Low</div>
<div galaxy:else>High</div>`

	result, _ := engine.Render(template, nil)

	if !strings.Contains(result, "Low") {
		t.Errorf("expected Low, got: %s", result)
	}
}

func TestComparisonGreaterEqual(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Set("score", int64(80))

	engine := NewEngine(ctx)
	template := `<div galaxy:if={score >= 80}>Pass</div>
<div galaxy:else>Fail</div>`

	result, _ := engine.Render(template, nil)

	if !strings.Contains(result, "Pass") {
		t.Errorf("expected Pass, got: %s", result)
	}
}

func TestComparisonLessEqual(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Set("score", int64(60))

	engine := NewEngine(ctx)
	template := `<div galaxy:if={score <= 60}>Need Improvement</div>
<div galaxy:else>OK</div>`

	result, _ := engine.Render(template, nil)

	if !strings.Contains(result, "Need Improvement") {
		t.Errorf("expected Need Improvement, got: %s", result)
	}
}

func TestComparisonNumberLiteral(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Set("count", int64(5))

	engine := NewEngine(ctx)
	template := `<div galaxy:if={count > 3}>Many</div>
<div galaxy:else>Few</div>`

	result, _ := engine.Render(template, nil)

	if !strings.Contains(result, "Many") {
		t.Errorf("expected Many, got: %s", result)
	}
}

func TestComparisonWithElsif(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Set("grade", int64(75))

	engine := NewEngine(ctx)
	template := `<p galaxy:if={grade >= 90}>A</p>
<p galaxy:elsif={grade >= 80}>B</p>
<p galaxy:elsif={grade >= 70}>C</p>
<p galaxy:else>F</p>`

	result, _ := engine.Render(template, nil)

	if !strings.Contains(result, "<p>C</p>") {
		t.Errorf("expected C, got: %s", result)
	}
}

func TestStringComparison(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Set("role", "admin")

	engine := NewEngine(ctx)
	template := `<div galaxy:if={role == "admin"}>Admin Panel</div>
<div galaxy:elsif={role == "user"}>User Panel</div>
<div galaxy:else>Guest</div>`

	result, _ := engine.Render(template, nil)

	if !strings.Contains(result, "Admin Panel") {
		t.Errorf("expected Admin Panel, got: %s", result)
	}
}

func TestFloatComparison(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Set("price", 19.99)

	engine := NewEngine(ctx)
	template := `<div galaxy:if={price < 20}>Affordable</div>
<div galaxy:else>Expensive</div>`

	result, _ := engine.Render(template, nil)

	if !strings.Contains(result, "Affordable") {
		t.Errorf("expected Affordable, got: %s", result)
	}
}

func TestMixedTypeComparison(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Set("count", int64(5))

	engine := NewEngine(ctx)
	template := `<div galaxy:if={count == 5}>Five</div>
<div galaxy:else>Not Five</div>`

	result, _ := engine.Render(template, nil)

	if !strings.Contains(result, "Five") {
		t.Errorf("expected Five, got: %s", result)
	}
}
