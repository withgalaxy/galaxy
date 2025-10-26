package content

import (
	"fmt"
)

type StaticPath struct {
	Params map[string]string
	Props  map[string]interface{}
}

type GetStaticPathsFunc func() ([]StaticPath, error)

func GetStaticPathsFromCollection(collectionName string, collections *Collections, paramName string) ([]StaticPath, error) {
	entries, err := collections.GetCollection(collectionName)
	if err != nil {
		return nil, fmt.Errorf("get collection %s: %w", collectionName, err)
	}

	paths := make([]StaticPath, 0, len(entries))
	for _, entry := range entries {
		paths = append(paths, StaticPath{
			Params: map[string]string{
				paramName: entry.Slug,
			},
			Props: map[string]interface{}{
				"entry": entry,
			},
		})
	}

	return paths, nil
}

func GetStaticPathsFromEntries(entries []*Entry, paramName string) []StaticPath {
	paths := make([]StaticPath, 0, len(entries))
	for _, entry := range entries {
		paths = append(paths, StaticPath{
			Params: map[string]string{
				paramName: entry.Slug,
			},
			Props: map[string]interface{}{
				"entry": entry,
			},
		})
	}
	return paths
}
