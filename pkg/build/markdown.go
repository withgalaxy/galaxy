package build

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/withgalaxy/galaxy/pkg/compiler"
	"github.com/withgalaxy/galaxy/pkg/executor"
	"github.com/withgalaxy/galaxy/pkg/parser"
	"github.com/withgalaxy/galaxy/pkg/router"
)

func (b *SSGBuilder) buildMarkdownRoute(route *router.Route) error {
	content, err := os.ReadFile(route.FilePath)
	if err != nil {
		return err
	}

	isMDX := filepath.Ext(route.FilePath) == ".mdx"
	var html string

	if isMDX {
		mdxDoc, err := parser.ParseMDX(string(content))
		if err != nil {
			return err
		}

		html = b.processMDXComponents(mdxDoc)

		if mdxDoc.Layout != "" {
			html, err = b.applyLayout(mdxDoc.Layout, route.FilePath, mdxDoc.Frontmatter, html)
			if err != nil {
				return err
			}
		}
	} else {
		doc, err := parser.ParseMarkdownWithYAMLFrontmatter(string(content))
		if err != nil {
			return err
		}

		html = doc.HTML

		if doc.Layout != "" {
			html, err = b.applyLayout(doc.Layout, route.FilePath, doc.Frontmatter, html)
			if err != nil {
				return err
			}
		}
	}

	outPath := b.getOutputPath(route.Pattern)
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(outPath, []byte(html), 0644); err != nil {
		return err
	}

	fmt.Printf("  ✓ %s → %s\n", route.Pattern, outPath)
	return nil
}

func (b *SSGBuilder) applyLayout(layoutPath, routePath string, frontmatter map[string]interface{}, content string) (string, error) {
	if !filepath.IsAbs(layoutPath) {
		layoutPath = filepath.Join(filepath.Dir(routePath), layoutPath)
	}

	props := make(map[string]interface{})
	for k, v := range frontmatter {
		props[k] = v
	}
	props["content"] = content

	slots := map[string]string{
		"default": content,
	}

	b.Compiler.CollectedStyles = nil
	rendered, err := b.Compiler.Compile(layoutPath, props, slots)
	if err != nil {
		return "", fmt.Errorf("compile layout %s: %w", layoutPath, err)
	}

	return rendered, nil
}

func (b *SSGBuilder) processMDXComponents(doc *parser.MDXDocument) string {
	ctx := executor.NewContext()
	for k, v := range doc.Frontmatter {
		ctx.Set(k, v)
	}

	return b.Compiler.ProcessComponentTags(doc.HTML, ctx)
}

func (b *SSGBuilder) BuildMarkdownRoutes() error {
	resolver := compiler.NewComponentResolver(b.SrcDir, nil)
	b.Compiler.SetResolver(resolver)

	for _, route := range b.Router.Routes {
		if route.Type != router.RouteMarkdown {
			continue
		}

		if err := b.buildMarkdownRoute(route); err != nil {
			return fmt.Errorf("build markdown route %s: %w", route.Pattern, err)
		}
	}

	return nil
}
