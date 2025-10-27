package template

import (
	"strings"
	"testing"

	"github.com/withgalaxy/galaxy/pkg/executor"
)

func TestMapPropertyAccess(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Execute(`var user = map[string]string{"name": "Alice", "role": "admin"}`)

	engine := NewEngine(ctx)
	template := "<p>Name: {user.name}, Role: {user.role}</p>"

	result, err := engine.Render(template, nil)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	expected := "<p>Name: Alice, Role: admin</p>"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestMapIteration(t *testing.T) {
	ctx := executor.NewContext()
	posts := []map[string]interface{}{
		{"title": "First Post", "slug": "first-post"},
		{"title": "Second Post", "slug": "second-post"},
	}
	ctx.Set("posts", posts)

	engine := NewEngine(ctx)
	template := `<li galaxy:for={post in posts}>{post.title} - {post.slug}</li>`

	result, err := engine.Render(template, nil)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "First Post - first-post") {
		t.Errorf("expected 'First Post - first-post' in result, got: %s", result)
	}
	if !strings.Contains(result, "Second Post - second-post") {
		t.Errorf("expected 'Second Post - second-post' in result, got: %s", result)
	}
}

func TestMapWithIntValues(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Execute(`var product = map[string]int{"price": 99, "stock": 42}`)

	engine := NewEngine(ctx)
	template := "<p>Price: ${product.price}, Stock: {product.stock}</p>"

	result, err := engine.Render(template, nil)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "Price: $99") {
		t.Error("expected 'Price: $99' in result")
	}
	if !strings.Contains(result, "Stock: 42") {
		t.Error("expected 'Stock: 42' in result")
	}
}

func TestNestedMapAccess(t *testing.T) {
	ctx := executor.NewContext()
	items := []map[string]interface{}{
		{"name": "Item A", "price": "10"},
		{"name": "Item B", "price": "20"},
	}
	ctx.Set("items", items)

	engine := NewEngine(ctx)
	template := `<li galaxy:for={item in items}>{item.name}: ${item.price}</li>`

	result, err := engine.Render(template, nil)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "Item A: $10") {
		t.Errorf("expected 'Item A: $10' in result, got: %s", result)
	}
	if !strings.Contains(result, "Item B: $20") {
		t.Errorf("expected 'Item B: $20' in result, got: %s", result)
	}
}
