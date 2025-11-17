package lsp

import (
	"strings"

	"go.lsp.dev/protocol"
)

func (s *Server) getTOMLCompletions(content string, pos protocol.Position) (*protocol.CompletionList, error) {
	schema := BuildTOMLSchema()
	ctx := DetectTOMLContext(content, pos)

	items := make([]protocol.CompletionItem, 0)

	if ctx.IsTable {
		items = append(items, getTableCompletions(schema)...)
	} else if ctx.IsValue {
		items = append(items, getValueCompletions(schema, ctx)...)
	} else if ctx.IsKey {
		items = append(items, getKeyCompletions(schema, ctx, content)...)
	} else {
		items = append(items, getTopLevelCompletions(schema)...)
	}

	return &protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

func getTableCompletions(schema *TOMLSchema) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0)

	tables := []string{"output", "server", "adapter", "security", "lifecycle", "markdown", "content"}

	for _, table := range tables {
		if tableSchema, ok := schema.Tables[table]; ok {
			items = append(items, protocol.CompletionItem{
				Label:  table,
				Kind:   protocol.CompletionItemKindModule,
				Detail: tableSchema.Description,
			})
		}
	}

	items = append(items, protocol.CompletionItem{
		Label:  "plugins",
		Kind:   protocol.CompletionItemKindModule,
		Detail: "Plugin configuration (use [[plugins]] for array)",
	})

	items = append(items, protocol.CompletionItem{
		Label:  "security.headers",
		Kind:   protocol.CompletionItemKindModule,
		Detail: "Security headers configuration",
	})

	items = append(items, protocol.CompletionItem{
		Label:  "security.bodyLimit",
		Kind:   protocol.CompletionItemKindModule,
		Detail: "Request body size limits",
	})

	return items
}

func getKeyCompletions(schema *TOMLSchema, ctx TOMLContext, content string) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0)

	tableSchema, ok := schema.Tables[ctx.Table]
	if !ok {
		return items
	}

	usedKeys := extractUsedKeys(content, ctx.Table)

	for fieldName, fieldSchema := range tableSchema.Fields {
		if usedKeys[fieldName] {
			continue
		}

		detail := fieldSchema.Type
		if fieldSchema.Default != "" {
			detail += " (default: " + fieldSchema.Default + ")"
		}

		insertText := fieldName + " = "
		if fieldSchema.Type == "string" || len(fieldSchema.EnumValues) > 0 {
			insertText += "\"\""
		} else if fieldSchema.Type == "bool" {
			insertText += "false"
		} else if fieldSchema.Type == "int" {
			insertText += "0"
		}

		items = append(items, protocol.CompletionItem{
			Label:      fieldName,
			Kind:       protocol.CompletionItemKindProperty,
			Detail:     detail,
			InsertText: insertText,
		})
	}

	return items
}

func getValueCompletions(schema *TOMLSchema, ctx TOMLContext) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0)

	fieldSchema, ok := schema.GetFieldSchema(ctx.Table, ctx.Key)
	if !ok {
		return items
	}

	if len(fieldSchema.EnumValues) > 0 {
		for _, value := range fieldSchema.EnumValues {
			documentation := fieldSchema.Description
			if ctx.Table == "output" && ctx.Key == "type" {
				switch value {
				case "static":
					documentation = "Static site generation - all pages rendered at build time"
				case "server":
					documentation = "Server-side rendering - pages rendered on each request"
				case "hybrid":
					documentation = "Hybrid mode - mix of static and server-rendered pages"
				}
			} else if ctx.Table == "adapter" && ctx.Key == "name" {
				switch value {
				case "standalone":
					documentation = "Standalone server (Docker, Railway, Fly.io) - supports all output types"
				case "cloudflare":
					documentation = "Cloudflare Pages - static only"
				case "netlify":
					documentation = "Netlify - static only"
				case "vercel":
					documentation = "Vercel - static only"
				}
			}

			items = append(items, protocol.CompletionItem{
				Label:         value,
				Kind:          protocol.CompletionItemKindValue,
				Detail:        fieldSchema.Type,
				Documentation: documentation,
			})
		}
	} else if fieldSchema.Type == "bool" {
		items = append(items,
			protocol.CompletionItem{
				Label:  "true",
				Kind:   protocol.CompletionItemKindValue,
				Detail: fieldSchema.Description,
			},
			protocol.CompletionItem{
				Label:  "false",
				Kind:   protocol.CompletionItemKindValue,
				Detail: fieldSchema.Description,
			},
		)
	}

	return items
}

func getTopLevelCompletions(schema *TOMLSchema) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0)

	rootTable, ok := schema.Tables[""]
	if !ok {
		return items
	}

	for fieldName, fieldSchema := range rootTable.Fields {
		if fieldSchema.IsTable {
			insertText := "[" + fieldName + "]"
			items = append(items, protocol.CompletionItem{
				Label:      fieldName,
				Kind:       protocol.CompletionItemKindModule,
				Detail:     fieldSchema.Description,
				InsertText: insertText,
			})
		} else {
			detail := fieldSchema.Type
			if fieldSchema.Default != "" {
				detail += " (default: " + fieldSchema.Default + ")"
			}

			insertText := fieldName + " = "
			if fieldSchema.Type == "string" || len(fieldSchema.EnumValues) > 0 {
				insertText += "\"\""
			}

			items = append(items, protocol.CompletionItem{
				Label:      fieldName,
				Kind:       protocol.CompletionItemKindProperty,
				Detail:     detail,
				InsertText: insertText,
			})
		}
	}

	return items
}

func extractUsedKeys(content string, table string) map[string]bool {
	used := make(map[string]bool)
	lines := strings.Split(content, "\n")

	inTargetTable := table == ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "[") && strings.Contains(trimmed, "]") {
			start := strings.Index(trimmed, "[")
			end := strings.Index(trimmed, "]")
			if start != -1 && end > start {
				currentTable := strings.TrimSpace(trimmed[start+1 : end])
				currentTable = strings.ReplaceAll(currentTable, "[[", "")
				currentTable = strings.ReplaceAll(currentTable, "]]", "")
				inTargetTable = currentTable == table
			}
			continue
		}

		if inTargetTable && strings.Contains(trimmed, "=") && !strings.HasPrefix(trimmed, "#") {
			parts := strings.SplitN(trimmed, "=", 2)
			key := strings.TrimSpace(parts[0])
			if key != "" {
				used[key] = true
			}
		}
	}

	return used
}
