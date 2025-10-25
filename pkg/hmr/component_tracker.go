package hmr

import (
	"crypto/sha256"
	"fmt"
	"sync"
)

type ComponentUsage struct {
	ComponentPath string
	UsedBy        map[string]bool
	LastHash      string
}

type ComponentTracker struct {
	mu         sync.RWMutex
	components map[string]*ComponentUsage
	pages      map[string][]string
}

func NewComponentTracker() *ComponentTracker {
	return &ComponentTracker{
		components: make(map[string]*ComponentUsage),
		pages:      make(map[string][]string),
	}
}

func (ct *ComponentTracker) TrackPageComponents(pagePath string, componentPaths []string) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.pages[pagePath] = componentPaths

	for _, compPath := range componentPaths {
		if _, exists := ct.components[compPath]; !exists {
			ct.components[compPath] = &ComponentUsage{
				ComponentPath: compPath,
				UsedBy:        make(map[string]bool),
			}
		}
		ct.components[compPath].UsedBy[pagePath] = true
	}
}

func (ct *ComponentTracker) UpdateComponentHash(componentPath, hash string) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	if comp, exists := ct.components[componentPath]; exists {
		comp.LastHash = hash
	}
}

func (ct *ComponentTracker) GetAffectedPages(componentPath string) []string {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	comp, exists := ct.components[componentPath]
	if !exists {
		return nil
	}

	pages := make([]string, 0, len(comp.UsedBy))
	for page := range comp.UsedBy {
		pages = append(pages, page)
	}
	return pages
}

func (ct *ComponentTracker) GetPageComponents(pagePath string) []string {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	return ct.pages[pagePath]
}

func (ct *ComponentTracker) IsComponentChanged(componentPath, currentContent string) bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	comp, exists := ct.components[componentPath]
	if !exists {
		return true
	}

	currentHash := hashContent(currentContent)
	return comp.LastHash != currentHash
}

func (ct *ComponentTracker) Clear() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.components = make(map[string]*ComponentUsage)
	ct.pages = make(map[string][]string)
}

func hashContent(content string) string {
	h := sha256.New()
	h.Write([]byte(content))
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}
