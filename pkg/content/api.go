package content

import (
	"fmt"
	"github.com/withgalaxy/galaxy/pkg/executor"
	"os"
	"path/filepath"
)

func init() {
	// Register as Galaxy.Content.Get for consistency
	executor.RegisterGlobalFunc("Galaxy.Content", "Get", func(args ...interface{}) (interface{}, error) {
		if len(args) < 2 {
			return nil, nil
		}

		collectionName, ok1 := args[0].(string)
		slug, ok2 := args[1].(string)
		if !ok1 || !ok2 {
			return nil, nil
		}

		contentDir := findContentDir()
		api := &ContentAPI{collections: NewCollections(contentDir)}
		result := api.Get(collectionName, slug)

		if result == nil {
			// Entry not found - return error to indicate 404
			return nil, fmt.Errorf("content entry not found: %s/%s (searched in: %s)", collectionName, slug, contentDir)
		}

		return result, nil
	})

	executor.RegisterGlobalFunc("Galaxy.Content", "GetCollection", func(args ...interface{}) (interface{}, error) {
		if len(args) < 1 {
			return nil, nil
		}

		collectionName, ok := args[0].(string)
		if !ok {
			return nil, nil
		}

		api := &ContentAPI{collections: NewCollections(findContentDir())}
		return api.GetCollection(collectionName), nil
	})
}

type ContentAPI struct {
	collections    *Collections
	shouldRedirect func(url string, status int)
}

// Get is a helper function for use in codegen mode
func Get(collectionName, slug string) map[string]interface{} {
	api := &ContentAPI{collections: NewCollections(findContentDir())}
	return api.Get(collectionName, slug)
}

// GetCollection is a helper function for use in codegen mode
func GetCollection(collectionName string) []map[string]interface{} {
	api := &ContentAPI{collections: NewCollections(findContentDir())}
	return api.GetCollection(collectionName)
}

func NewContentAPI(redirectFunc func(string, int)) *ContentAPI {
	contentDir := findContentDir()

	return &ContentAPI{
		collections:    NewCollections(contentDir),
		shouldRedirect: redirectFunc,
	}
}

func findContentDir() string {
	candidates := []string{
		"src/content",
		"./src/content",
		"../../src/content",
	}

	for _, dir := range candidates {
		if _, err := os.Stat(dir); err == nil {
			absPath, _ := filepath.Abs(dir)
			return absPath
		}
	}

	return "src/content"
}

func (c *ContentAPI) Get(collectionName, slug string) map[string]interface{} {
	entry, err := c.collections.GetEntry(collectionName, slug)
	if err != nil {
		if c.shouldRedirect != nil {
			c.shouldRedirect("/404", 404)
		}
		return nil
	}

	rendered, _ := c.collections.Render(entry)

	result := make(map[string]interface{})

	for k, v := range entry.Data {
		result[k] = v
	}

	result["slug"] = entry.Slug
	result["content"] = rendered
	result["body"] = entry.Body

	return result
}

func (c *ContentAPI) GetCollection(collectionName string) []map[string]interface{} {
	entries, err := c.collections.GetCollection(collectionName)
	if err != nil {
		return nil
	}

	var result []map[string]interface{}
	for _, entry := range entries {
		item := make(map[string]interface{})

		for k, v := range entry.Data {
			item[k] = v
		}

		item["slug"] = entry.Slug

		result = append(result, item)
	}

	return result
}
