package build

import (
	"path/filepath"
	"testing"
)

func TestMakePathRelative(t *testing.T) {
	tmpDir := t.TempDir()
	
	builder := &SSGBuilder{
		OutDir: tmpDir,
	}

	tests := []struct {
		name          string
		assetPath     string
		htmlFilePath  string
		expectedPath  string
	}{
		{
			name:         "root index to assets",
			assetPath:    "/_assets/styles.css",
			htmlFilePath: filepath.Join(tmpDir, "index.html"),
			expectedPath: "_assets/styles.css",
		},
		{
			name:         "one level deep",
			assetPath:    "/_assets/styles.css",
			htmlFilePath: filepath.Join(tmpDir, "about", "index.html"),
			expectedPath: "../_assets/styles.css",
		},
		{
			name:         "two levels deep",
			assetPath:    "/_assets/styles.css",
			htmlFilePath: filepath.Join(tmpDir, "blog", "post", "index.html"),
			expectedPath: "../../_assets/styles.css",
		},
		{
			name:         "empty path returns empty",
			assetPath:    "",
			htmlFilePath: filepath.Join(tmpDir, "index.html"),
			expectedPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.makePathRelative(tt.assetPath, tt.htmlFilePath)
			if result != tt.expectedPath {
				t.Errorf("makePathRelative() = %v, want %v", result, tt.expectedPath)
			}
		})
	}
}

func TestDetectCollectionFromFrontmatter(t *testing.T) {
	builder := &SSGBuilder{}

	tests := []struct {
		name           string
		frontmatter    string
		paramName      string
		expectedResult string
	}{
		{
			name:           "detect blog collection",
			frontmatter:    `entry := Galaxy.Content.Get("blog", Galaxy.Params["slug"])`,
			paramName:      "slug",
			expectedResult: "blog",
		},
		{
			name:           "detect posts collection",
			frontmatter:    `data := Galaxy.Content.Get("posts", Galaxy.Params["id"])`,
			paramName:      "id",
			expectedResult: "posts",
		},
		{
			name:           "no collection detected",
			frontmatter:    `var title = "Test"`,
			paramName:      "slug",
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.detectCollectionFromFrontmatter(tt.frontmatter, tt.paramName)
			if result != tt.expectedResult {
				t.Errorf("detectCollectionFromFrontmatter() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestGetOutputPath(t *testing.T) {
	tmpDir := t.TempDir()
	builder := &SSGBuilder{
		OutDir: tmpDir,
	}

	tests := []struct {
		name         string
		pattern      string
		expectedPath string
	}{
		{
			name:         "root pattern",
			pattern:      "/",
			expectedPath: filepath.Join(tmpDir, "index.html"),
		},
		{
			name:         "about page",
			pattern:      "/about",
			expectedPath: filepath.Join(tmpDir, "about", "index.html"),
		},
		{
			name:         "nested page",
			pattern:      "/blog/post",
			expectedPath: filepath.Join(tmpDir, "blog", "post", "index.html"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.getOutputPath(tt.pattern)
			if result != tt.expectedPath {
				t.Errorf("getOutputPath() = %v, want %v", result, tt.expectedPath)
			}
		})
	}
}
