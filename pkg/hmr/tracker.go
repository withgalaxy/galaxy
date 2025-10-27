package hmr

import (
	"os"
	"sync"

	"github.com/withgalaxy/galaxy/pkg/parser"
)

type ChangeTracker struct {
	cache map[string]*parser.Component
	mu    sync.RWMutex
}

func NewChangeTracker() *ChangeTracker {
	return &ChangeTracker{
		cache: make(map[string]*parser.Component),
	}
}

func (t *ChangeTracker) DetectChange(filePath string) (*ComponentDiff, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	newComp, err := parser.Parse(string(content))
	if err != nil {
		return nil, err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	oldComp, exists := t.cache[filePath]
	if !exists {
		t.cache[filePath] = newComp
		// First time seeing this file - treat as full change to trigger update
		diff := ComponentDiff{
			TemplateChanged:    true,
			StylesChanged:      len(newComp.Styles) > 0,
			ScriptsChanged:     len(newComp.Scripts) > 0,
			FrontmatterChanged: newComp.Frontmatter != "",
		}
		return &diff, nil
	}

	diff := DiffComponents(oldComp, newComp)
	t.cache[filePath] = newComp

	return &diff, nil
}

func (t *ChangeTracker) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cache = make(map[string]*parser.Component)
}
