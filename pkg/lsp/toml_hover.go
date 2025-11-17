package lsp

import (
	"fmt"
	"strings"

	"go.lsp.dev/protocol"
)

func (s *Server) getTOMLHover(content string, pos protocol.Position) (*protocol.Hover, error) {
	schema := BuildTOMLSchema()
	ctx := DetectTOMLContext(content, pos)

	if ctx.IsValue || ctx.IsKey {
		if fieldSchema, ok := schema.GetFieldSchema(ctx.Table, ctx.Key); ok {
			return &protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.Markdown,
					Value: formatFieldDocumentation(ctx.Table, ctx.Key, fieldSchema),
				},
			}, nil
		}
	}

	if ctx.IsTable {
		if tableSchema, ok := schema.GetTableSchema(ctx.Table); ok {
			return &protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.Markdown,
					Value: formatTableDocumentation(ctx.Table, tableSchema),
				},
			}, nil
		}
	}

	return nil, nil
}

func formatFieldDocumentation(table, key string, field FieldSchema) string {
	var doc strings.Builder

	doc.WriteString(fmt.Sprintf("### %s.%s\n\n", table, key))
	doc.WriteString(fmt.Sprintf("**Type:** `%s`", field.Type))

	if len(field.EnumValues) > 0 {
		doc.WriteString(" (enum)\n\n")
		doc.WriteString("**Valid values:**\n")
		for _, val := range field.EnumValues {
			doc.WriteString(fmt.Sprintf("- `%s`", val))

			if table == "output" && key == "type" {
				switch val {
				case "static":
					doc.WriteString(" - Static site generation (all pages at build time)")
				case "server":
					doc.WriteString(" - Server-side rendering (pages on each request)")
				case "hybrid":
					doc.WriteString(" - Mix of static and server-rendered pages")
				}
			} else if table == "adapter" && key == "name" {
				switch val {
				case "standalone":
					doc.WriteString(" - Self-contained server (Docker, Railway, Fly.io)")
				case "cloudflare":
					doc.WriteString(" - Cloudflare Pages (static only)")
				case "netlify":
					doc.WriteString(" - Netlify (static only)")
				case "vercel":
					doc.WriteString(" - Vercel (static only)")
				}
			}
			doc.WriteString("\n")
		}
	} else {
		doc.WriteString("\n\n")
	}

	if field.Default != "" {
		doc.WriteString(fmt.Sprintf("**Default:** `%s`\n\n", field.Default))
	}

	if field.Description != "" {
		doc.WriteString(field.Description)
		doc.WriteString("\n")
	}

	if table == "adapter" && key == "name" {
		doc.WriteString("\n**Note:** `vercel`, `netlify`, and `cloudflare` adapters only support `output.type = \"static\"`. Use `standalone` for SSR/hybrid.")
	}

	if table == "output" && key == "type" {
		doc.WriteString("\n**Note:** Server and hybrid modes require an adapter (e.g., `adapter.name = \"standalone\"`).")
	}

	return doc.String()
}

func formatTableDocumentation(table string, tableSchema TableSchema) string {
	var doc strings.Builder

	doc.WriteString(fmt.Sprintf("### [%s]\n\n", table))

	if tableSchema.Description != "" {
		doc.WriteString(tableSchema.Description)
		doc.WriteString("\n\n")
	}

	doc.WriteString("**Available fields:**\n")
	for fieldName, fieldSchema := range tableSchema.Fields {
		doc.WriteString(fmt.Sprintf("- `%s` (%s)", fieldName, fieldSchema.Type))
		if fieldSchema.Default != "" {
			doc.WriteString(fmt.Sprintf(" - default: `%s`", fieldSchema.Default))
		}
		doc.WriteString("\n")
	}

	return doc.String()
}
