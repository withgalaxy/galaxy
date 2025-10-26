package content

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestContent(t *testing.T) (string, func()) {
	tmpDir := t.TempDir()

	blogDir := filepath.Join(tmpDir, "blog")
	if err := os.MkdirAll(blogDir, 0755); err != nil {
		t.Fatal(err)
	}

	post1 := `---
title: "First Post"
author: "Alice"
draft: false
---

# First Post Content

This is the first post.`

	post2 := `---
title: "Second Post"
author: "Bob"
draft: true
---

# Second Post

Draft content.`

	if err := os.WriteFile(filepath.Join(blogDir, "post-1.md"), []byte(post1), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(blogDir, "post-2.md"), []byte(post2), 0644); err != nil {
		t.Fatal(err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestGetCollection(t *testing.T) {
	contentDir, cleanup := setupTestContent(t)
	defer cleanup()

	collections := NewCollections(contentDir)

	entries, err := collections.GetCollection("blog")
	if err != nil {
		t.Fatalf("Failed to get collection: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}

	for _, entry := range entries {
		if entry.Slug != "post-1" && entry.Slug != "post-2" {
			t.Errorf("Unexpected slug: %s", entry.Slug)
		}

		if entry.Collection != "blog" {
			t.Errorf("Expected collection 'blog', got %s", entry.Collection)
		}
	}
}

func TestGetEntry(t *testing.T) {
	contentDir, cleanup := setupTestContent(t)
	defer cleanup()

	collections := NewCollections(contentDir)

	entry, err := collections.GetEntry("blog", "post-1")
	if err != nil {
		t.Fatalf("Failed to get entry: %v", err)
	}

	if entry.Slug != "post-1" {
		t.Errorf("Expected slug 'post-1', got %s", entry.Slug)
	}

	title := entry.GetString("title")
	if title != "First Post" {
		t.Errorf("Expected title 'First Post', got %s", title)
	}

	author := entry.GetString("author")
	if author != "Alice" {
		t.Errorf("Expected author 'Alice', got %s", author)
	}

	draft := entry.GetBool("draft")
	if draft {
		t.Errorf("Expected draft to be false")
	}
}

func TestGetEntryNotFound(t *testing.T) {
	contentDir, cleanup := setupTestContent(t)
	defer cleanup()

	collections := NewCollections(contentDir)

	_, err := collections.GetEntry("blog", "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent entry")
	}
}

func TestRender(t *testing.T) {
	contentDir, cleanup := setupTestContent(t)
	defer cleanup()

	collections := NewCollections(contentDir)

	entry, err := collections.GetEntry("blog", "post-1")
	if err != nil {
		t.Fatalf("Failed to get entry: %v", err)
	}

	html, err := collections.Render(entry)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	if html == "" {
		t.Error("Expected non-empty HTML")
	}
}

func TestGetStaticPathsFromCollection(t *testing.T) {
	contentDir, cleanup := setupTestContent(t)
	defer cleanup()

	collections := NewCollections(contentDir)

	paths, err := GetStaticPathsFromCollection("blog", collections, "slug")
	if err != nil {
		t.Fatalf("Failed to get static paths: %v", err)
	}

	if len(paths) != 2 {
		t.Errorf("Expected 2 paths, got %d", len(paths))
	}

	for _, path := range paths {
		slug, ok := path.Params["slug"]
		if !ok {
			t.Error("Expected slug param")
		}

		if slug != "post-1" && slug != "post-2" {
			t.Errorf("Unexpected slug: %s", slug)
		}

		entry, ok := path.Props["entry"]
		if !ok {
			t.Error("Expected entry prop")
		}

		if entry == nil {
			t.Error("Entry should not be nil")
		}
	}
}
