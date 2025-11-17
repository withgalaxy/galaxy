package lsp

import (
	"strings"

	"go.lsp.dev/protocol"
)

type TOMLContext struct {
	Table      string
	Key        string
	IsValue    bool
	IsKey      bool
	IsTable    bool
	LinePrefix string
}

func DetectTOMLContext(content string, pos protocol.Position) TOMLContext {
	lines := strings.Split(content, "\n")
	if int(pos.Line) >= len(lines) {
		return TOMLContext{}
	}

	currentLine := lines[pos.Line]
	if int(pos.Character) > len(currentLine) {
		return TOMLContext{}
	}

	beforeCursor := currentLine[:pos.Character]
	ctx := TOMLContext{
		Table:      findCurrentTable(lines, int(pos.Line)),
		LinePrefix: strings.TrimSpace(beforeCursor),
	}

	trimmedLine := strings.TrimSpace(currentLine)

	if strings.HasPrefix(trimmedLine, "[") && !strings.Contains(beforeCursor, "]") {
		ctx.IsTable = true
		tableStart := strings.Index(beforeCursor, "[")
		if tableStart != -1 {
			ctx.Table = strings.TrimSpace(beforeCursor[tableStart+1:])
		}
		return ctx
	}

	if strings.Contains(beforeCursor, "=") {
		ctx.IsValue = true
		parts := strings.SplitN(beforeCursor, "=", 2)
		ctx.Key = strings.TrimSpace(parts[0])

		afterEquals := ""
		if len(parts) > 1 {
			afterEquals = strings.TrimSpace(parts[1])
		}
		if strings.HasPrefix(afterEquals, "\"") && !strings.HasSuffix(afterEquals, "\"") {
			ctx.IsValue = true
		}
		return ctx
	}

	if trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") && !strings.HasPrefix(trimmedLine, "[") {
		ctx.IsKey = true
		ctx.Key = strings.TrimSpace(beforeCursor)
		return ctx
	}

	return ctx
}

func findCurrentTable(lines []string, currentLine int) string {
	table := ""
	for i := currentLine; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "[") && strings.Contains(line, "]") {
			start := strings.Index(line, "[")
			end := strings.Index(line, "]")
			if start != -1 && end > start {
				table = strings.TrimSpace(line[start+1 : end])
				table = strings.ReplaceAll(table, "[[", "")
				table = strings.ReplaceAll(table, "]]", "")
				break
			}
		}
	}
	return table
}

func IsTOMLFile(uri protocol.DocumentURI) bool {
	uriStr := string(uri)
	return strings.HasSuffix(uriStr, ".toml") || strings.Contains(uriStr, "galaxy.config.toml")
}
