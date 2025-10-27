package template

import (
	"strings"
	"testing"

	"github.com/withgalaxy/galaxy/pkg/executor"
)

func TestNestedTagsInForDirective(t *testing.T) {
	ctx := executor.NewContext()
	posts := []map[string]interface{}{
		{"title": "First Post", "slug": "first-post"},
		{"title": "Second Post", "slug": "second-post"},
	}
	ctx.Set("posts", posts)

	engine := NewEngine(ctx)
	template := `<article galaxy:for={post in posts}><h2>{post.title}</h2><a href="/blog/{post.slug}">Read</a></article>`

	result, err := engine.Render(template, nil)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "<h2>First Post</h2>") {
		t.Errorf("expected h2 with First Post, got: %s", result)
	}
	if !strings.Contains(result, `<a href="/blog/first-post">Read</a>`) {
		t.Errorf("expected link to first-post, got: %s", result)
	}
	if !strings.Contains(result, "<h2>Second Post</h2>") {
		t.Errorf("expected h2 with Second Post, got: %s", result)
	}
}

func TestDeeplyNestedTags(t *testing.T) {
	ctx := executor.NewContext()
	items := []map[string]interface{}{
		{"name": "Item A", "price": "10"},
	}
	ctx.Set("items", items)

	engine := NewEngine(ctx)
	template := `<div galaxy:for={item in items}><section><article><h3>{item.name}</h3><p>${item.price}</p></article></section></div>`

	result, err := engine.Render(template, nil)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "<h3>Item A</h3>") {
		t.Errorf("expected h3, got: %s", result)
	}
	if !strings.Contains(result, "<p>$10</p>") {
		t.Errorf("expected p tag, got: %s", result)
	}
}

func TestSameTagNested(t *testing.T) {
	ctx := executor.NewContext()
	items := []interface{}{"A", "B"}
	ctx.Set("items", items)

	engine := NewEngine(ctx)
	template := `<div galaxy:for={item in items}><div class="inner">{item}</div></div>`

	result, err := engine.Render(template, nil)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, `<div class="inner">A</div>`) {
		t.Errorf("expected inner div A, got: %s", result)
	}
	if !strings.Contains(result, `<div class="inner">B</div>`) {
		t.Errorf("expected inner div B, got: %s", result)
	}
}

func TestMultipleDirectivesInTemplate(t *testing.T) {
	ctx := executor.NewContext()
	posts := []interface{}{"Post 1", "Post 2"}
	comments := []interface{}{"Comment 1"}
	ctx.Set("posts", posts)
	ctx.Set("comments", comments)

	engine := NewEngine(ctx)
	template := `<div><li galaxy:for={post in posts}>{post}</li><span galaxy:for={comment in comments}>{comment}</span></div>`

	result, err := engine.Render(template, nil)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "<li>Post 1</li>") {
		t.Errorf("expected Post 1, got: %s", result)
	}
	if !strings.Contains(result, "<li>Post 2</li>") {
		t.Errorf("expected Post 2, got: %s", result)
	}
	if !strings.Contains(result, "<span>Comment 1</span>") {
		t.Errorf("expected Comment 1, got: %s", result)
	}
}

func TestNestedIfDirective(t *testing.T) {
	ctx := executor.NewContext()
	ctx.Execute(`var show = 1`)

	engine := NewEngine(ctx)
	template := `<div galaxy:if={show}><h1>Title</h1><p>Content</p></div>`

	result, err := engine.Render(template, nil)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "<h1>Title</h1>") {
		t.Errorf("expected h1, got: %s", result)
	}
	if !strings.Contains(result, "<p>Content</p>") {
		t.Errorf("expected p, got: %s", result)
	}
}

func TestForWithAdditionalAttributes(t *testing.T) {
	ctx := executor.NewContext()
	items := []interface{}{"A", "B"}
	ctx.Set("items", items)

	engine := NewEngine(ctx)
	template := `<div galaxy:for={item in items} class="item" data-id="123"><span>{item}</span></div>`

	result, err := engine.Render(template, nil)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, `class="item"`) {
		t.Errorf("expected class attribute, got: %s", result)
	}
	if !strings.Contains(result, `data-id="123"`) {
		t.Errorf("expected data-id attribute, got: %s", result)
	}
	if !strings.Contains(result, "<span>A</span>") {
		t.Errorf("expected span A, got: %s", result)
	}
}

func TestMultiLevelNestedFieldAccess(t *testing.T) {
	type User struct {
		Name  string
		Email string
	}
	type Project struct {
		Name  string
		Owner User
	}

	ctx := executor.NewContext()
	project := &Project{
		Name:  "Galaxy Framework",
		Owner: User{Name: "John Doe", Email: "john@example.com"},
	}
	ctx.Set("project", project)

	engine := NewEngine(ctx)
	template := `<div><h1>{project.Name}</h1><p>Owner: {project.Owner.Name}</p><small>{project.Owner.Email}</small></div>`

	result, err := engine.Render(template, nil)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "<h1>Galaxy Framework</h1>") {
		t.Errorf("expected project name, got: %s", result)
	}
	if !strings.Contains(result, "<p>Owner: John Doe</p>") {
		t.Errorf("expected owner name, got: %s", result)
	}
	if !strings.Contains(result, "<small>john@example.com</small>") {
		t.Errorf("expected owner email, got: %s", result)
	}
}

func TestForLoopWithCustomStructSlice(t *testing.T) {
	type Task struct {
		Title  string
		Status string
	}

	ctx := executor.NewContext()
	tasks := []Task{
		{Title: "Build feature", Status: "in-progress"},
		{Title: "Write tests", Status: "pending"},
	}
	ctx.Set("tasks", tasks)

	engine := NewEngine(ctx)
	template := `<ul><li galaxy:for={task in tasks}>{task.Title} - {task.Status}</li></ul>`

	result, err := engine.Render(template, nil)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "<li>Build feature - in-progress</li>") {
		t.Errorf("expected first task, got: %s", result)
	}
	if !strings.Contains(result, "<li>Write tests - pending</li>") {
		t.Errorf("expected second task, got: %s", result)
	}
}

func TestForLoopWithPointerStructSlice(t *testing.T) {
	type Product struct {
		Name  string
		Price int
	}

	ctx := executor.NewContext()
	products := []*Product{
		{Name: "Laptop", Price: 999},
		{Name: "Mouse", Price: 25},
	}
	ctx.Set("products", products)

	engine := NewEngine(ctx)
	template := `<div galaxy:for={p in products}><span>{p.Name}</span> - <strong>${p.Price}</strong></div>`

	result, err := engine.Render(template, nil)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "<span>Laptop</span> - <strong>$999</strong>") {
		t.Errorf("expected laptop product, got: %s", result)
	}
	if !strings.Contains(result, "<span>Mouse</span> - <strong>$25</strong>") {
		t.Errorf("expected mouse product, got: %s", result)
	}
}

func TestForLoopWithNestedStructAccess(t *testing.T) {
	type User struct {
		Name string
	}
	type Task struct {
		Title    string
		Assignee User
	}
	type TaskData struct {
		Task     Task
		Priority string
	}

	ctx := executor.NewContext()
	taskList := []TaskData{
		{Task: Task{Title: "Fix bug", Assignee: User{Name: "Alice"}}, Priority: "high"},
		{Task: Task{Title: "Add docs", Assignee: User{Name: "Bob"}}, Priority: "low"},
	}
	ctx.Set("taskList", taskList)

	engine := NewEngine(ctx)
	template := `<table><tr galaxy:for={td in taskList}><td>{td.Task.Title}</td><td>{td.Task.Assignee.Name}</td><td>{td.Priority}</td></tr></table>`

	result, err := engine.Render(template, nil)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(result, "<td>Fix bug</td>") {
		t.Errorf("expected task title, got: %s", result)
	}
	if !strings.Contains(result, "<td>Alice</td>") {
		t.Errorf("expected assignee name, got: %s", result)
	}
	if !strings.Contains(result, "<td>high</td>") {
		t.Errorf("expected priority, got: %s", result)
	}
	if !strings.Contains(result, "<td>Add docs</td>") {
		t.Errorf("expected second task title, got: %s", result)
	}
	if !strings.Contains(result, "<td>Bob</td>") {
		t.Errorf("expected second assignee name, got: %s", result)
	}
}
