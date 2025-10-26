package content

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cameron-webmatter/galaxy/pkg/parser"
)

type Collections struct {
	ContentDir string
	configs    map[string]CollectionConfig
	cache      map[string][]*Entry
}

func NewCollections(contentDir string) *Collections {
	return &Collections{
		ContentDir: contentDir,
		configs:    make(map[string]CollectionConfig),
		cache:      make(map[string][]*Entry),
	}
}

func (c *Collections) DefineCollection(name string, config CollectionConfig) {
	c.configs[name] = config
}

func (c *Collections) GetCollection(name string) ([]*Entry, error) {
	if cached, ok := c.cache[name]; ok {
		return cached, nil
	}

	collectionDir := filepath.Join(c.ContentDir, name)
	if _, err := os.Stat(collectionDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("collection %s not found at %s", name, collectionDir)
	}

	var entries []*Entry

	err := filepath.Walk(collectionDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".mdx") {
			return nil
		}

		entry, err := c.parseEntry(path, name)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}

		entries = append(entries, entry)
		return nil
	})

	if err != nil {
		return nil, err
	}

	c.cache[name] = entries
	return entries, nil
}

func (c *Collections) GetEntry(collectionName, slug string) (*Entry, error) {
	entries, err := c.GetCollection(collectionName)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.Slug == slug {
			return entry, nil
		}
	}

	return nil, fmt.Errorf("entry %s not found in collection %s", slug, collectionName)
}

func (c *Collections) parseEntry(filePath, collectionName string) (*Entry, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	doc, err := parser.ParseMarkdownWithYAMLFrontmatter(string(content))
	if err != nil {
		return nil, err
	}

	filename := filepath.Base(filePath)
	slug := strings.TrimSuffix(filename, filepath.Ext(filename))

	entry := &Entry{
		ID:         filepath.ToSlash(filePath),
		Slug:       slug,
		Collection: collectionName,
		Data:       doc.Frontmatter,
		Body:       doc.Content,
		FilePath:   filePath,
		RawContent: string(content),
	}

	return entry, nil
}

func (c *Collections) Render(entry *Entry) (string, error) {
	doc, err := parser.ParseMarkdownWithYAMLFrontmatter(entry.RawContent)
	if err != nil {
		return "", err
	}

	return doc.HTML, nil
}

func (c *Collections) ClearCache() {
	c.cache = make(map[string][]*Entry)
}
