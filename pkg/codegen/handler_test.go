package codegen

import (
	"testing"

	"github.com/withgalaxy/galaxy/pkg/router"
)

func TestTransformContentEntryAccess(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "transform entry.title",
			input:    `var title = entry.title`,
			expected: `var title = entry["title"]`,
		},
		{
			name:     "transform entry.content",
			input:    `var content = entry.content`,
			expected: `var content = entry["content"]`,
		},
		{
			name:     "transform entry.pubDate",
			input:    `var date = entry.pubDate`,
			expected: `var date = entry["pubDate"]`,
		},
		{
			name:     "transform entry.author",
			input:    `var author = entry.author`,
			expected: `var author = entry["author"]`,
		},
		{
			name:     "transform entry.slug",
			input:    `var slug = entry.slug`,
			expected: `var slug = entry["slug"]`,
		},
		{
			name:     "transform entry.body",
			input:    `var body = entry.body`,
			expected: `var body = entry["body"]`,
		},
		{
			name:     "transform post.title",
			input:    `var title = post.title`,
			expected: `var title = post["title"]`,
		},
		{
			name:     "no transformation for method calls",
			input:    `entry.GetString("title")`,
			expected: `entry.GetString("title")`,
		},
		{
			name:     "multiple transformations",
			input:    `var title = entry.title; var date = entry.pubDate`,
			expected: `var title = entry["title"]; var date = entry["pubDate"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transformContentEntryAccess(tt.input)
			if result != tt.expected {
				t.Errorf("transformContentEntryAccess() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestTransformCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "transform Galaxy.Content.Get",
			input:    `entry := Galaxy.Content.Get("blog", slug)`,
			expected: `entry := content.Get("blog", slug)`,
		},
		{
			name:     "transform Galaxy.Content.GetCollection",
			input:    `posts := Galaxy.Content.GetCollection("blog")`,
			expected: `posts := content.GetCollection("blog")`,
		},
		{
			name:     "transform Galaxy.Locals access",
			input:    `val := Galaxy.Locals.myValue`,
			expected: `val := locals["myValue"]`,
		},
		{
			name:     "transform Locals access",
			input:    `val := Locals.myValue`,
			expected: `val := locals["myValue"]`,
		},
		{
			name:     "combined transformations",
			input:    `entry := Galaxy.Content.Get("blog", slug); var title = entry.title`,
			expected: `entry := content.Get("blog", slug); var title = entry["title"]`,
		},
	}

	// Create a mock generator with a simple route
	gen := &HandlerGenerator{
		Route: &router.Route{Pattern: "/test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.transformCode(tt.input)
			if result != tt.expected {
				t.Errorf("transformCode() = %q, want %q", result, tt.expected)
			}
		})
	}
}
