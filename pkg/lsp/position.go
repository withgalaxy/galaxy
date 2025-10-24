package lsp

import (
	"strings"

	"github.com/cameron-webmatter/galaxy/pkg/parser"
	"go.lsp.dev/protocol"
)

// PositionMapper maps between .gxc and virtual .go positions
type PositionMapper struct {
	frontmatterStart int // Line number where frontmatter starts (after first ---)
	frontmatterEnd   int // Line number where frontmatter ends (second ---)
	importBlockStart int // Line in .gxc where import block starts
	importBlockEnd   int // Line in .gxc where import block ends
	importLineCount  int // Number of import lines
	content          string
}

// NewPositionMapper analyzes gxc content
func NewPositionMapper(content string) *PositionMapper {
	lines := strings.Split(content, "\n")
	pm := &PositionMapper{
		frontmatterStart: -1,
		frontmatterEnd:   -1,
		importBlockStart: -1,
		importBlockEnd:   -1,
		content:          content,
	}

	inFrontmatter := false
	inImportBlock := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				pm.frontmatterStart = i + 1
			} else {
				pm.frontmatterEnd = i
				break
			}
		} else if inFrontmatter {
			if strings.HasPrefix(trimmed, "import (") {
				inImportBlock = true
				pm.importBlockStart = i
			} else if inImportBlock && strings.Contains(trimmed, ")") {
				pm.importBlockEnd = i
				inImportBlock = false
			} else if inImportBlock {
				pm.importLineCount++
			} else if strings.HasPrefix(trimmed, "import ") {
				// Single import line
				if pm.importBlockStart == -1 {
					pm.importBlockStart = i
				}
				pm.importBlockEnd = i
				pm.importLineCount++
			}
		}
	}

	return pm
}

// GxcToGo converts .gxc position to virtual .go position
func (pm *PositionMapper) GxcToGo(line, char int) (int, int) {
	if pm.frontmatterStart == -1 {
		return line, char
	}

	// Adjust for:
	// - package main
	// - empty line
	// - import block
	// - empty line
	// - func _gxcPage() {

	offset := 1 // package main
	offset += 1 // empty line

	if pm.importLineCount > 0 {
		offset += 2 + pm.importLineCount // import ( ... )
		offset += 1                      // empty line
	}

	offset += 1 // func _gxcPage() {

	gxcLine := line - pm.frontmatterStart
	goLine := gxcLine + offset

	return goLine, char
}

// GoToGxc converts virtual .go position back to .gxc position
func (pm *PositionMapper) GoToGxc(line, char int) (int, int) {
	if pm.frontmatterStart == -1 {
		return line, char
	}

	// Calculate offset (same as GxcToGo)
	offset := 1 // package main
	offset += 1 // empty line

	if pm.importLineCount > 0 {
		offset += 2 + pm.importLineCount // import ( ... )
		offset += 1                      // empty line
	}

	offset += 1 // func _gxcPage() {

	// Reverse the mapping: goLine = gxcLine + offset
	// So: gxcLine = goLine - offset
	gxcLine := line - offset
	absoluteLine := gxcLine + pm.frontmatterStart

	return absoluteLine, char
}

// TransformPosition transforms a protocol.Position from .go to .gxc
func (pm *PositionMapper) TransformPosition(pos protocol.Position) protocol.Position {
	line, char := pm.GoToGxc(int(pos.Line), int(pos.Character))
	return protocol.Position{
		Line:      uint32(line),
		Character: uint32(char),
	}
}

// TransformRange transforms a protocol.Range from .go to .gxc
func (pm *PositionMapper) TransformRange(r protocol.Range) protocol.Range {
	return protocol.Range{
		Start: pm.TransformPosition(r.Start),
		End:   pm.TransformPosition(r.End),
	}
}

// TransformTextEdit transforms a protocol.TextEdit from .go to .gxc
func (pm *PositionMapper) TransformTextEdit(edit protocol.TextEdit) protocol.TextEdit {
	return protocol.TextEdit{
		Range:   pm.TransformRange(edit.Range),
		NewText: edit.NewText,
	}
}

// TransformCompletionItem transforms all position-related fields in a CompletionItem
func (pm *PositionMapper) TransformCompletionItem(item protocol.CompletionItem) protocol.CompletionItem {
	// Transform TextEdit if present
	if item.TextEdit != nil {
		transformed := pm.TransformTextEdit(*item.TextEdit)
		item.TextEdit = &transformed
	}

	// Transform AdditionalTextEdits if present - these are usually imports
	if len(item.AdditionalTextEdits) > 0 {
		transformed := make([]protocol.TextEdit, 0, len(item.AdditionalTextEdits))
		for _, edit := range item.AdditionalTextEdits {
			// Check if this is an import addition
			if pm.isImportEdit(edit) {
				// Map to gxc import location
				gxcEdit := pm.transformImportEdit(edit)
				if gxcEdit != nil {
					transformed = append(transformed, *gxcEdit)
				}
			} else {
				transformed = append(transformed, pm.TransformTextEdit(edit))
			}
		}
		item.AdditionalTextEdits = transformed
	}

	return item
}

// isImportEdit checks if a TextEdit is adding an import
func (pm *PositionMapper) isImportEdit(edit protocol.TextEdit) bool {
	return strings.Contains(edit.NewText, "import")
}

// transformImportEdit transforms an import addition from .go to .gxc
func (pm *PositionMapper) transformImportEdit(edit protocol.TextEdit) *protocol.TextEdit {
	// Extract import path from edit
	importLine := strings.TrimSpace(edit.NewText)

	// Determine where to insert in .gxc
	if pm.importBlockStart != -1 {
		// Import block exists - add to end of block
		insertLine := pm.importBlockEnd
		if pm.importBlockEnd > pm.importBlockStart+1 {
			// Multi-line import block - insert before closing )
			return &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: uint32(insertLine), Character: 0},
					End:   protocol.Position{Line: uint32(insertLine), Character: 0},
				},
				NewText: "\t" + importLine + "\n",
			}
		} else {
			// Single import - add after
			return &protocol.TextEdit{
				Range: protocol.Range{
					Start: protocol.Position{Line: uint32(insertLine + 1), Character: 0},
					End:   protocol.Position{Line: uint32(insertLine + 1), Character: 0},
				},
				NewText: importLine + "\n",
			}
		}
	} else if pm.frontmatterStart != -1 {
		// No import block - create one at start of frontmatter
		return &protocol.TextEdit{
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(pm.frontmatterStart), Character: 0},
				End:   protocol.Position{Line: uint32(pm.frontmatterStart), Character: 0},
			},
			NewText: importLine + "\n\n",
		}
	}

	return nil
}

// ScriptPositionMapper maps between .gxc script positions and virtual .go positions
type ScriptPositionMapper struct {
	scriptStart     int // Line number where script content starts (after <script>)
	importLineCount int // Number of import lines in script
}

// NewScriptPositionMapper creates a mapper for script tags
func NewScriptPositionMapper(scriptContent string, scriptStart int) *ScriptPositionMapper {
	spm := &ScriptPositionMapper{
		scriptStart: scriptStart,
	}

	// Count import lines
	lines := strings.Split(scriptContent, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "import") {
			spm.importLineCount++
		}
	}

	return spm
}

// GxcToGo converts .gxc script position to virtual .go position
func (spm *ScriptPositionMapper) GxcToGo(line, char int) (int, int) {
	// Calculate offset for virtual Go file:
	// - package main (line 0)
	// - empty line (line 1)
	// - import block (if imports exist)
	// - empty line
	// - func _gxcScript() { (line N)

	offset := 1 // package main
	offset += 1 // empty line

	if spm.importLineCount > 0 {
		offset += 1 + spm.importLineCount // import lines (no wrapping parens for script)
		offset += 1                       // empty line
	}

	offset += 1 // func _gxcScript() {

	// Convert absolute .gxc line to script-relative line
	scriptLine := line - spm.scriptStart
	goLine := scriptLine + offset

	return goLine, char
}

// GoToGxc converts virtual .go position back to .gxc script position
func (spm *ScriptPositionMapper) GoToGxc(line, char int) (int, int) {
	// Calculate offset (same as GxcToGo)
	offset := 1 // package main
	offset += 1 // empty line

	if spm.importLineCount > 0 {
		offset += 1 + spm.importLineCount
		offset += 1
	}

	offset += 1 // func _gxcScript() {

	// Reverse the mapping
	scriptLine := line - offset
	absoluteLine := scriptLine + spm.scriptStart

	return absoluteLine, char
}

// TransformPosition transforms a protocol.Position from .go to .gxc
func (spm *ScriptPositionMapper) TransformPosition(pos protocol.Position) protocol.Position {
	line, char := spm.GoToGxc(int(pos.Line), int(pos.Character))
	return protocol.Position{
		Line:      uint32(line),
		Character: uint32(char),
	}
}

// TransformRange transforms a protocol.Range from .go to .gxc
func (spm *ScriptPositionMapper) TransformRange(r protocol.Range) protocol.Range {
	return protocol.Range{
		Start: spm.TransformPosition(r.Start),
		End:   spm.TransformPosition(r.End),
	}
}

// TransformTextEdit transforms a protocol.TextEdit from .go to .gxc
func (spm *ScriptPositionMapper) TransformTextEdit(edit protocol.TextEdit) protocol.TextEdit {
	return protocol.TextEdit{
		Range:   spm.TransformRange(edit.Range),
		NewText: edit.NewText,
	}
}

// TransformCompletionItem transforms all position-related fields in a CompletionItem
func (spm *ScriptPositionMapper) TransformCompletionItem(item protocol.CompletionItem) protocol.CompletionItem {
	// Transform TextEdit if present
	if item.TextEdit != nil {
		transformed := spm.TransformTextEdit(*item.TextEdit)
		item.TextEdit = &transformed
	}

	// Transform AdditionalTextEdits if present
	if len(item.AdditionalTextEdits) > 0 {
		transformed := make([]protocol.TextEdit, len(item.AdditionalTextEdits))
		for i, edit := range item.AdditionalTextEdits {
			transformed[i] = spm.TransformTextEdit(edit)
		}
		item.AdditionalTextEdits = transformed
	}

	return item
}

// IsInFrontmatter checks if position is within frontmatter region
func IsInFrontmatter(pos protocol.Position, content string) bool {
	comp, err := parser.Parse(content)
	if err != nil || comp.Frontmatter == "" {
		return false
	}

	lines := strings.Split(content, "\n")
	if int(pos.Line) >= len(lines) {
		return false
	}

	// Find frontmatter boundaries
	start := -1
	end := -1
	dashCount := 0

	for i, line := range lines {
		if strings.TrimSpace(line) == "---" {
			dashCount++
			if dashCount == 1 {
				start = i
			} else if dashCount == 2 {
				end = i
				break
			}
		}
	}

	if start == -1 || end == -1 {
		return false
	}

	return int(pos.Line) > start && int(pos.Line) < end
}

// IsInScript checks if position is within a <script> tag
func IsInScript(pos protocol.Position, content string) bool {
	lines := strings.Split(content, "\n")
	if int(pos.Line) >= len(lines) {
		return false
	}

	// Find script tag boundaries
	inScript := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "<script") {
			inScript = true
		}

		if inScript && i == int(pos.Line) {
			// Don't include the <script> line itself
			if !strings.HasPrefix(trimmed, "<script") && !strings.HasPrefix(trimmed, "</script>") {
				return true
			}
		}

		if strings.HasPrefix(trimmed, "</script>") {
			inScript = false
		}
	}

	return false
}

// FindScriptAtPosition finds the script content and boundaries for the cursor position
func FindScriptAtPosition(content string, pos protocol.Position) (scriptContent string, startLine int, endLine int, found bool) {
	lines := strings.Split(content, "\n")
	if int(pos.Line) >= len(lines) {
		return "", 0, 0, false
	}

	scriptStart := -1
	scriptEnd := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "<script") {
			scriptStart = i + 1 // Line after <script>
		}

		if strings.HasPrefix(trimmed, "</script>") {
			scriptEnd = i // Line before </script>

			// Check if cursor is in this script block
			if scriptStart != -1 && int(pos.Line) > scriptStart-1 && int(pos.Line) < scriptEnd+1 {
				// Extract script content
				scriptLines := lines[scriptStart:scriptEnd]
				scriptContent = strings.Join(scriptLines, "\n")
				return scriptContent, scriptStart, scriptEnd, true
			}

			scriptStart = -1
		}
	}

	return "", 0, 0, false
}
