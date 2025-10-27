package hmr

import (
	"testing"
)

func TestNewComponentTracker(t *testing.T) {
	ct := NewComponentTracker()
	if ct == nil {
		t.Fatal("NewComponentTracker returned nil")
	}
	if ct.components == nil {
		t.Error("components map not initialized")
	}
	if ct.pages == nil {
		t.Error("pages map not initialized")
	}
}

func TestComponentTracker_TrackPageComponents(t *testing.T) {
	ct := NewComponentTracker()

	pagePath := "/pages/index.gxc"
	components := []string{
		"/components/Header.gxc",
		"/components/Footer.gxc",
	}

	ct.TrackPageComponents(pagePath, components)

	if len(ct.pages) != 1 {
		t.Errorf("expected 1 page, got %d", len(ct.pages))
	}

	if len(ct.components) != 2 {
		t.Errorf("expected 2 components, got %d", len(ct.components))
	}

	headerComp := ct.components["/components/Header.gxc"]
	if headerComp == nil {
		t.Fatal("Header component not tracked")
	}

	if !headerComp.UsedBy[pagePath] {
		t.Error("Header component should be used by index page")
	}
}

func TestComponentTracker_TrackMultiplePages(t *testing.T) {
	ct := NewComponentTracker()

	ct.TrackPageComponents("/pages/index.gxc", []string{"/components/Header.gxc"})
	ct.TrackPageComponents("/pages/about.gxc", []string{"/components/Header.gxc"})

	headerComp := ct.components["/components/Header.gxc"]
	if len(headerComp.UsedBy) != 2 {
		t.Errorf("expected Header used by 2 pages, got %d", len(headerComp.UsedBy))
	}

	if !headerComp.UsedBy["/pages/index.gxc"] {
		t.Error("Header should be used by index page")
	}
	if !headerComp.UsedBy["/pages/about.gxc"] {
		t.Error("Header should be used by about page")
	}
}

func TestComponentTracker_UpdateComponentHash(t *testing.T) {
	ct := NewComponentTracker()

	compPath := "/components/Button.gxc"
	ct.TrackPageComponents("/pages/index.gxc", []string{compPath})

	hash := "abc123def456"
	ct.UpdateComponentHash(compPath, hash)

	comp := ct.components[compPath]
	if comp.LastHash != hash {
		t.Errorf("expected hash %s, got %s", hash, comp.LastHash)
	}
}

func TestComponentTracker_UpdateComponentHash_Nonexistent(t *testing.T) {
	ct := NewComponentTracker()

	ct.UpdateComponentHash("/nonexistent.gxc", "hash")

	if len(ct.components) != 0 {
		t.Error("should not create component when updating nonexistent")
	}
}

func TestComponentTracker_GetAffectedPages(t *testing.T) {
	ct := NewComponentTracker()

	compPath := "/components/Nav.gxc"
	ct.TrackPageComponents("/pages/index.gxc", []string{compPath})
	ct.TrackPageComponents("/pages/about.gxc", []string{compPath})
	ct.TrackPageComponents("/pages/contact.gxc", []string{compPath})

	affected := ct.GetAffectedPages(compPath)

	if len(affected) != 3 {
		t.Errorf("expected 3 affected pages, got %d", len(affected))
	}

	pageMap := make(map[string]bool)
	for _, page := range affected {
		pageMap[page] = true
	}

	if !pageMap["/pages/index.gxc"] {
		t.Error("index.gxc should be affected")
	}
	if !pageMap["/pages/about.gxc"] {
		t.Error("about.gxc should be affected")
	}
	if !pageMap["/pages/contact.gxc"] {
		t.Error("contact.gxc should be affected")
	}
}

func TestComponentTracker_GetAffectedPages_Nonexistent(t *testing.T) {
	ct := NewComponentTracker()

	affected := ct.GetAffectedPages("/nonexistent.gxc")

	if affected != nil {
		t.Errorf("expected nil for nonexistent component, got %v", affected)
	}
}

func TestComponentTracker_GetPageComponents(t *testing.T) {
	ct := NewComponentTracker()

	pagePath := "/pages/index.gxc"
	components := []string{
		"/components/Header.gxc",
		"/components/Footer.gxc",
		"/components/Button.gxc",
	}

	ct.TrackPageComponents(pagePath, components)

	result := ct.GetPageComponents(pagePath)

	if len(result) != 3 {
		t.Errorf("expected 3 components, got %d", len(result))
	}

	compMap := make(map[string]bool)
	for _, comp := range result {
		compMap[comp] = true
	}

	for _, comp := range components {
		if !compMap[comp] {
			t.Errorf("component %s not found in result", comp)
		}
	}
}

func TestComponentTracker_GetPageComponents_Nonexistent(t *testing.T) {
	ct := NewComponentTracker()

	result := ct.GetPageComponents("/nonexistent.gxc")

	if result != nil {
		t.Errorf("expected nil for nonexistent page, got %v", result)
	}
}

func TestComponentTracker_IsComponentChanged_New(t *testing.T) {
	ct := NewComponentTracker()

	changed := ct.IsComponentChanged("/new.gxc", "content")

	if !changed {
		t.Error("new component should be considered changed")
	}
}

func TestComponentTracker_IsComponentChanged_Same(t *testing.T) {
	ct := NewComponentTracker()

	compPath := "/components/Button.gxc"
	content := "<button>Click me</button>"

	ct.TrackPageComponents("/pages/index.gxc", []string{compPath})
	hash := hashContent(content)
	ct.UpdateComponentHash(compPath, hash)

	changed := ct.IsComponentChanged(compPath, content)

	if changed {
		t.Error("component with same content should not be changed")
	}
}

func TestComponentTracker_IsComponentChanged_Different(t *testing.T) {
	ct := NewComponentTracker()

	compPath := "/components/Button.gxc"
	oldContent := "<button>Click me</button>"
	newContent := "<button>Click here</button>"

	ct.TrackPageComponents("/pages/index.gxc", []string{compPath})
	hash := hashContent(oldContent)
	ct.UpdateComponentHash(compPath, hash)

	changed := ct.IsComponentChanged(compPath, newContent)

	if !changed {
		t.Error("component with different content should be changed")
	}
}

func TestComponentTracker_Clear(t *testing.T) {
	ct := NewComponentTracker()

	ct.TrackPageComponents("/pages/index.gxc", []string{"/components/Header.gxc"})
	ct.TrackPageComponents("/pages/about.gxc", []string{"/components/Footer.gxc"})

	if len(ct.components) == 0 {
		t.Error("components should not be empty before clear")
	}
	if len(ct.pages) == 0 {
		t.Error("pages should not be empty before clear")
	}

	ct.Clear()

	if len(ct.components) != 0 {
		t.Errorf("components should be empty after clear, got %d", len(ct.components))
	}
	if len(ct.pages) != 0 {
		t.Errorf("pages should be empty after clear, got %d", len(ct.pages))
	}
}

func TestComponentTracker_OverwritePageTracking(t *testing.T) {
	ct := NewComponentTracker()

	pagePath := "/pages/index.gxc"
	ct.TrackPageComponents(pagePath, []string{"/components/Header.gxc"})
	ct.TrackPageComponents(pagePath, []string{"/components/Footer.gxc"})

	components := ct.GetPageComponents(pagePath)
	if len(components) != 1 {
		t.Errorf("expected 1 component after overwrite, got %d", len(components))
	}
	if components[0] != "/components/Footer.gxc" {
		t.Error("should have Footer component after overwrite")
	}
}

func TestComponentTracker_SharedComponent(t *testing.T) {
	ct := NewComponentTracker()

	sharedComp := "/components/Layout.gxc"
	ct.TrackPageComponents("/pages/index.gxc", []string{sharedComp, "/components/Hero.gxc"})
	ct.TrackPageComponents("/pages/about.gxc", []string{sharedComp, "/components/Team.gxc"})
	ct.TrackPageComponents("/pages/contact.gxc", []string{sharedComp, "/components/Form.gxc"})

	affected := ct.GetAffectedPages(sharedComp)
	if len(affected) != 3 {
		t.Errorf("shared component should affect 3 pages, got %d", len(affected))
	}

	affected = ct.GetAffectedPages("/components/Hero.gxc")
	if len(affected) != 1 {
		t.Errorf("Hero component should affect 1 page, got %d", len(affected))
	}
}

func TestHashContent(t *testing.T) {
	content1 := "test content"
	content2 := "test content"
	content3 := "different content"

	hash1 := hashContent(content1)
	hash2 := hashContent(content2)
	hash3 := hashContent(content3)

	if hash1 != hash2 {
		t.Error("same content should produce same hash")
	}
	if hash1 == hash3 {
		t.Error("different content should produce different hash")
	}
	if len(hash1) != 16 {
		t.Errorf("hash should be 16 chars, got %d", len(hash1))
	}
}

func TestComponentTracker_Concurrent(t *testing.T) {
	ct := NewComponentTracker()

	done := make(chan bool, 10)

	for i := 0; i < 5; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				ct.TrackPageComponents("/pages/test.gxc", []string{"/components/C.gxc"})
			}
			done <- true
		}(i)
	}

	for i := 0; i < 5; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				_ = ct.GetAffectedPages("/components/C.gxc")
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	affected := ct.GetAffectedPages("/components/C.gxc")
	if len(affected) != 1 {
		t.Errorf("expected 1 affected page after concurrent operations, got %d", len(affected))
	}
}
