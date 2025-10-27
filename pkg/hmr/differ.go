package hmr

import (
	"crypto/sha256"
	"fmt"

	"github.com/withgalaxy/galaxy/pkg/parser"
)

type ComponentDiff struct {
	TemplateChanged    bool
	StylesChanged      bool
	ScriptsChanged     bool
	FrontmatterChanged bool
	ChangedStyles      []int
	ChangedScripts     []int
}

func DiffComponents(old, new *parser.Component) ComponentDiff {
	diff := ComponentDiff{
		ChangedStyles:  make([]int, 0),
		ChangedScripts: make([]int, 0),
	}

	diff.FrontmatterChanged = hashString(old.Frontmatter) != hashString(new.Frontmatter)
	diff.TemplateChanged = hashString(old.Template) != hashString(new.Template)

	oldStyleMap := make(map[string]int)
	for i, style := range old.Styles {
		hash := hashString(style.Content)
		oldStyleMap[hash] = i
	}

	for i, style := range new.Styles {
		hash := hashString(style.Content)
		if _, exists := oldStyleMap[hash]; !exists {
			diff.StylesChanged = true
			diff.ChangedStyles = append(diff.ChangedStyles, i)
		}
	}

	if len(old.Styles) != len(new.Styles) {
		diff.StylesChanged = true
	}

	oldScriptMap := make(map[string]int)
	for i, script := range old.Scripts {
		hash := hashString(script.Content)
		oldScriptMap[hash] = i
	}

	for i, script := range new.Scripts {
		hash := hashString(script.Content)
		if _, exists := oldScriptMap[hash]; !exists {
			diff.ScriptsChanged = true
			diff.ChangedScripts = append(diff.ChangedScripts, i)
		}
	}

	if len(old.Scripts) != len(new.Scripts) {
		diff.ScriptsChanged = true
	}

	return diff
}

func (d ComponentDiff) NeedsFullReload() bool {
	return d.FrontmatterChanged || d.ScriptsChanged
}

func (d ComponentDiff) CanHotSwapStyles() bool {
	return d.StylesChanged && !d.TemplateChanged && !d.ScriptsChanged && !d.FrontmatterChanged
}

func hashString(s string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(s)))
}
