package lsp

import (
	"testing"

	"go.lsp.dev/protocol"
)

func TestDetectDefinitionContext(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		line     uint32
		char     uint32
		expected DefinitionContext
		compName string
		propName string
		varName  string
	}{
		{
			name:     "cursor on component name",
			content:  `<Nav userName={user} />`,
			line:     0,
			char:     2,
			expected: ContextComponentName,
			compName: "Nav",
		},
		{
			name:     "cursor on prop name",
			content:  `<Nav userName={user} />`,
			line:     0,
			char:     7,
			expected: ContextPropName,
			compName: "Nav",
			propName: "userName",
		},
		{
			name:     "cursor on variable in prop value",
			content:  `<Nav userName={user} />`,
			line:     0,
			char:     16,
			expected: ContextVariableName,
			varName:  "user",
		},
		{
			name:     "cursor on whitespace returns component",
			content:  `<Nav userName={user} />`,
			line:     0,
			char:     4,
			expected: ContextComponentName,
			compName: "Nav",
		},
		{
			name:     "cursor on HTML tag returns none",
			content:  `<div class="foo">`,
			line:     0,
			char:     2,
			expected: ContextNone,
		},
		{
			name:     "cursor on prop with string value",
			content:  `<Nav title="Test" />`,
			line:     0,
			char:     6,
			expected: ContextPropName,
			compName: "Nav",
			propName: "title",
		},
		{
			name:     "multiline component - prop on second attribute",
			content:  `<Nav userName={user} age={age} />`,
			line:     0,
			char:     22,
			expected: ContextPropName,
			compName: "Nav",
			propName: "age",
		},
		{
			name:     "variable with field access",
			content:  `<Nav userName={user.name} />`,
			line:     0,
			char:     16,
			expected: ContextVariableName,
			varName:  "user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := detectDefinitionContext(tt.content, protocol.Position{
				Line:      tt.line,
				Character: tt.char,
			})

			if target.Context != tt.expected {
				t.Errorf("expected context %v, got %v", tt.expected, target.Context)
			}

			if tt.compName != "" && target.ComponentName != tt.compName {
				t.Errorf("expected component name %q, got %q", tt.compName, target.ComponentName)
			}

			if tt.propName != "" && target.PropName != tt.propName {
				t.Errorf("expected prop name %q, got %q", tt.propName, target.PropName)
			}

			if tt.varName != "" && target.VariableName != tt.varName {
				t.Errorf("expected variable name %q, got %q", tt.varName, target.VariableName)
			}
		})
	}
}

func TestExtractVariableFromPropValue(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		cursorOffset int
		expected     string
	}{
		{
			name:         "simple variable",
			value:        "user",
			cursorOffset: 2,
			expected:     "user",
		},
		{
			name:         "field access",
			value:        "user.name",
			cursorOffset: 2,
			expected:     "user",
		},
		{
			name:         "cursor on field name",
			value:        "user.name",
			cursorOffset: 7,
			expected:     "name",
		},
		{
			name:         "array access",
			value:        "items[0]",
			cursorOffset: 3,
			expected:     "items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVariableFromPropValue(tt.value, tt.cursorOffset)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFindVarInCode(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		varName  string
		expected int
	}{
		{
			name: "var with type",
			code: `var userName string = "John"
var age int = 30`,
			varName:  "userName",
			expected: 0,
		},
		{
			name: "var without type",
			code: `userName := "John"
age := 30`,
			varName:  "userName",
			expected: 0,
		},
		{
			name: "second variable",
			code: `var userName string = "John"
var age int = 30`,
			varName:  "age",
			expected: 1,
		},
		{
			name: "variable not found",
			code: `var userName string = "John"
var age int = 30`,
			varName:  "missing",
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findVarInCode(tt.code, tt.varName)
			if result != tt.expected {
				t.Errorf("expected line %d, got %d", tt.expected, result)
			}
		})
	}
}
