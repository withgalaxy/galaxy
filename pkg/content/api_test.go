package content

import (
	"os"
	"path/filepath"
	"testing"
)

func TestContentAPIGet(t *testing.T) {
	// Create temp content directory
	tmpDir := t.TempDir()
	contentDir := filepath.Join(tmpDir, "content", "blog")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a test markdown file
	mdContent := `---
title: Test Post
author: Test Author
pubDate: 2024-01-01
---

# Hello World

This is test content.
`
	mdFile := filepath.Join(contentDir, "test-post.md")
	if err := os.WriteFile(mdFile, []byte(mdContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test Get function
	collections := NewCollections(filepath.Join(tmpDir, "content"))
	api := &ContentAPI{collections: collections}
	
	result := api.Get("blog", "test-post")
	
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	
	if result["title"] != "Test Post" {
		t.Errorf("Expected title 'Test Post', got %v", result["title"])
	}
	
	if result["author"] != "Test Author" {
		t.Errorf("Expected author 'Test Author', got %v", result["author"])
	}
	
	if result["slug"] != "test-post" {
		t.Errorf("Expected slug 'test-post', got %v", result["slug"])
	}
	
	if result["content"] == nil {
		t.Error("Expected rendered content, got nil")
	}
}

func TestContentAPIGetCollection(t *testing.T) {
	// Create temp content directory
	tmpDir := t.TempDir()
	contentDir := filepath.Join(tmpDir, "content", "posts")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create multiple test markdown files
	posts := []struct {
		slug  string
		title string
	}{
		{"first-post", "First Post"},
		{"second-post", "Second Post"},
		{"third-post", "Third Post"},
	}

	for _, post := range posts {
		content := "---\ntitle: " + post.title + "\n---\n\nContent here."
		file := filepath.Join(contentDir, post.slug+".md")
		if err := os.WriteFile(file, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Test GetCollection function
	collections := NewCollections(filepath.Join(tmpDir, "content"))
	api := &ContentAPI{collections: collections}
	
	result := api.GetCollection("posts")
	
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	
	if len(result) != 3 {
		t.Errorf("Expected 3 posts, got %d", len(result))
	}
	
	// Check that all posts have required fields
	for _, entry := range result {
		if entry["title"] == nil {
			t.Error("Entry missing title field")
		}
		if entry["slug"] == nil {
			t.Error("Entry missing slug field")
		}
	}
}

func TestContentAPIGetNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	collections := NewCollections(filepath.Join(tmpDir, "content"))
	api := &ContentAPI{collections: collections}
	
	result := api.Get("blog", "non-existent")
	
	if result != nil {
		t.Errorf("Expected nil for non-existent entry, got %v", result)
	}
}

func TestFindContentDir(t *testing.T) {
	// This test verifies that findContentDir returns a reasonable default
	result := findContentDir()
	
	if result == "" {
		t.Error("Expected non-empty content directory path")
	}
}
