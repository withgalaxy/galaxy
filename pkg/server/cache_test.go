package server

import (
	"net/http"
	"testing"
)

func TestNewPageCache(t *testing.T) {
	cache := NewPageCache()
	if cache == nil {
		t.Fatal("NewPageCache returned nil")
	}
	if cache.pages == nil {
		t.Error("pages map not initialized")
	}
}

func TestPageCache_SetAndGet(t *testing.T) {
	cache := NewPageCache()

	plugin := &PagePlugin{
		Template:        "<h1>Test</h1>",
		FrontmatterHash: "hash123",
		TemplateHash:    "hash456",
		PluginPath:      "/test/plugin.so",
	}

	cache.Set("/test", plugin)

	retrieved, ok := cache.Get("/test")
	if !ok {
		t.Fatal("expected to find cached plugin")
	}

	if retrieved.Template != plugin.Template {
		t.Errorf("expected template %s, got %s", plugin.Template, retrieved.Template)
	}
	if retrieved.FrontmatterHash != plugin.FrontmatterHash {
		t.Errorf("expected hash %s, got %s", plugin.FrontmatterHash, retrieved.FrontmatterHash)
	}
}

func TestPageCache_GetNotFound(t *testing.T) {
	cache := NewPageCache()

	_, ok := cache.Get("/nonexistent")
	if ok {
		t.Error("expected not to find plugin")
	}
}

func TestPageCache_Invalidate(t *testing.T) {
	cache := NewPageCache()

	plugin := &PagePlugin{Template: "<h1>Test</h1>"}
	cache.Set("/test", plugin)

	_, ok := cache.Get("/test")
	if !ok {
		t.Fatal("expected to find plugin before invalidate")
	}

	cache.Invalidate("/test")

	_, ok = cache.Get("/test")
	if ok {
		t.Error("expected not to find plugin after invalidate")
	}
}

func TestPageCache_OverwriteEntry(t *testing.T) {
	cache := NewPageCache()

	plugin1 := &PagePlugin{Template: "<h1>V1</h1>"}
	plugin2 := &PagePlugin{Template: "<h1>V2</h1>"}

	cache.Set("/test", plugin1)
	cache.Set("/test", plugin2)

	retrieved, ok := cache.Get("/test")
	if !ok {
		t.Fatal("expected to find plugin")
	}

	if retrieved.Template != "<h1>V2</h1>" {
		t.Error("expected second plugin to overwrite first")
	}
}

func TestPageCache_MultipleEntries(t *testing.T) {
	cache := NewPageCache()

	cache.Set("/page1", &PagePlugin{Template: "P1"})
	cache.Set("/page2", &PagePlugin{Template: "P2"})
	cache.Set("/page3", &PagePlugin{Template: "P3"})

	p1, ok1 := cache.Get("/page1")
	p2, ok2 := cache.Get("/page2")
	p3, ok3 := cache.Get("/page3")

	if !ok1 || !ok2 || !ok3 {
		t.Error("expected all entries to be found")
	}

	if p1.Template != "P1" || p2.Template != "P2" || p3.Template != "P3" {
		t.Error("templates don't match")
	}
}

func TestPageCache_ConcurrentAccess(t *testing.T) {
	cache := NewPageCache()
	done := make(chan bool, 20)

	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				cache.Set("/test", &PagePlugin{Template: "test"})
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				cache.Get("/test")
			}
			done <- true
		}(i)
	}

	for i := 0; i < 20; i++ {
		<-done
	}
}

func TestPagePlugin_WithHandler(t *testing.T) {
	handlerCalled := false

	plugin := &PagePlugin{
		Handler: func(w http.ResponseWriter, r *http.Request, params map[string]string, locals map[string]interface{}) {
			handlerCalled = true
		},
		Template: "<h1>Test</h1>",
	}

	if plugin.Handler == nil {
		t.Fatal("Handler not set")
	}

	plugin.Handler(nil, nil, nil, nil)

	if !handlerCalled {
		t.Error("expected handler to be called")
	}
}
