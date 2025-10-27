package orbit

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cameron-webmatter/galaxy/pkg/parser"
	"github.com/cameron-webmatter/galaxy/pkg/router"
)

func (p *GalaxyPlugin) handleMarkdown(w http.ResponseWriter, r *http.Request, route *router.Route, params map[string]string) {
	content, err := os.ReadFile(route.FilePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	doc, err := parser.ParseMarkdownWithYAMLFrontmatter(string(content))
	if err != nil {
		http.Error(w, fmt.Sprintf("Markdown parse error: %v", err), http.StatusInternalServerError)
		return
	}

	html := doc.HTML

	if doc.Layout != "" {
		layoutPath := filepath.Join(filepath.Dir(p.PagesDir), doc.Layout)
		if !filepath.IsAbs(doc.Layout) {
			layoutPath = filepath.Join(filepath.Dir(route.FilePath), doc.Layout)
		}

		props := make(map[string]interface{})
		for k, v := range doc.Frontmatter {
			props[k] = v
		}
		props["content"] = doc.HTML

		slots := map[string]string{
			"default": doc.HTML,
		}

		rendered, err := p.Compiler.Compile(layoutPath, props, slots)
		if err != nil {
			http.Error(w, fmt.Sprintf("Layout compile error: %v", err), http.StatusInternalServerError)
			return
		}

		html = rendered
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
