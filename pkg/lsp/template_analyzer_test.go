package lsp

import (
	"os"
	"testing"

	"go.lsp.dev/protocol"
)

func TestTemplateAnalyzer(t *testing.T) {
	// Create test component
	testCompDir := "/tmp/test-component/src/components"
	os.MkdirAll(testCompDir, 0755)

	// Also test with real taskflow components
	taskflowServer := &Server{
		cache:          make(map[protocol.DocumentURI]*DocumentState),
		componentCache: make(map[string]*ComponentInfo),
		rootPath:       "/Users/cameron/dev/galaxy-mono/taskflow",
	}

	server := &Server{
		cache:          make(map[protocol.DocumentURI]*DocumentState),
		componentCache: make(map[string]*ComponentInfo),
		rootPath:       "/tmp/test-component",
	}

	tests := []struct {
		name          string
		content       string
		expectedCount int
		expectedMsg   string
	}{
		{
			name: "undefined variable in expression",
			content: `---
userName := "Alice"
---
<div>
  <p>{title}</p>
</div>`,
			expectedCount: 1,
			expectedMsg:   "Undefined variable: title",
		},
		{
			name: "defined variable should not error",
			content: `---
userName := "Alice"
---
<div>
  <p>{userName}</p>
</div>`,
			expectedCount: 0,
		},
		{
			name: "undefined variable in galaxy:if",
			content: `---
isActive := true
---
<div>
  <p galaxy:if={showContent}>Test</p>
</div>`,
			expectedCount: 1,
			expectedMsg:   "Undefined variable in galaxy:if: showContent",
		},
		{
			name: "undefined variable in galaxy:for",
			content: `---
userName := "Alice"
---
<div>
  <div galaxy:for={item, idx in unknownList}>
    <span>{item}</span>
  </div>
</div>`,
			expectedCount: 1,
			expectedMsg:   "Undefined variable in galaxy:for: unknownList",
		},
		{
			name: "undefined variable in classList",
			content: `---
isActive := true
---
<div classList={{"active": undefinedVar}}>
  Test
</div>`,
			expectedCount: 1,
			expectedMsg:   "Undefined variable in classList: undefinedVar",
		},
		{
			name: "multiple errors",
			content: `---
userName := "Alice"
---
<div>
  <p>{missingVar1}</p>
  <p galaxy:if={missingVar2}>Test</p>
  <div classList={{"active": missingVar3}}>Test</div>
</div>`,
			expectedCount: 3,
		},
		{
			name: "Galaxy API should be available",
			content: `---
if Galaxy.Locals.user == nil {
    Galaxy.Redirect("/login", 302)
}
---
<div>
  <p>Test</p>
</div>`,
			expectedCount: 0,
		},
		{
			name: "component prop type mismatch",
			content: `---
userName := "Alice"
---
<TestComponent userName={123} />`,
			expectedCount: 1,
			expectedMsg:   "Type mismatch for prop 'userName': expected string, got int. Hint: convert with fmt.Sprintf",
		},
		{
			name: "real Nav component test",
			content: `---
userName := "Alice"
---
<Nav userName={1} />`,
			expectedCount: 1,
			expectedMsg:   "Type mismatch for prop 'userName': expected string, got int. Hint: convert with fmt.Sprintf",
		},
		{
			name: "variable type inference from scope",
			content: `---
var count int = 5
var name string = "test"
---
<TestComponent userName={count} />`,
			expectedCount: 1,
			expectedMsg:   "Type mismatch for prop 'userName': expected string, got int. Hint: convert with fmt.Sprintf",
		},
		{
			name: "correct variable type passes",
			content: `---
var name string = "Alice"
---
<TestComponent userName={name} />`,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use taskflowServer for the Nav component test
			testServer := server
			if tt.name == "real Nav component test" {
				testServer = taskflowServer
			}

			diagnostics := testServer.analyzeTemplate(tt.content)

			if len(diagnostics) != tt.expectedCount {
				t.Errorf("Expected %d diagnostics, got %d", tt.expectedCount, len(diagnostics))
				for i, d := range diagnostics {
					t.Logf("  [%d] %s: %s", i, d.Source, d.Message)
				}
			}

			if tt.expectedMsg != "" && len(diagnostics) > 0 {
				found := false
				for _, d := range diagnostics {
					if d.Message == tt.expectedMsg {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected message '%s' not found. Got:", tt.expectedMsg)
					for _, d := range diagnostics {
						t.Logf("  - %s", d.Message)
					}
				}
			}
		})
	}
}
