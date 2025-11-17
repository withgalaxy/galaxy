package lsp

import (
	"os"
	"strings"

	"github.com/withgalaxy/galaxy/pkg/config"
	"github.com/withgalaxy/galaxy/pkg/executor"
	"github.com/withgalaxy/galaxy/pkg/parser"
	"go.lsp.dev/protocol"
)

func (s *Server) analyze(content string) []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0)

	comp, err := parser.Parse(content)
	if err != nil {
		diagnostics = append(diagnostics, protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 1},
			},
			Severity: protocol.DiagnosticSeverityError,
			Source:   "gxc-parser",
			Message:  err.Error(),
		})
		return diagnostics
	}

	if comp.Frontmatter != "" {
		ctx := executor.NewContext()
		ctx.SetLocals(make(map[string]any))
		if err := ctx.Execute(comp.Frontmatter); err != nil {
			r := comp.FrontmatterRange
			diagnostics = append(diagnostics, protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: uint32(r.Start.Line - 1), Character: uint32(r.Start.Column)},
					End:   protocol.Position{Line: uint32(r.End.Line - 1), Character: uint32(r.End.Column)},
				},
				Severity: protocol.DiagnosticSeverityError,
				Source:   "gxc-executor",
				Message:  err.Error(),
			})
		}
	}

	// Validate template expressions and directives
	diagnostics = append(diagnostics, s.analyzeTemplate(content)...)

	return diagnostics
}

func (s *Server) analyzeTOML(content string) []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0)

	tmpFile, err := os.CreateTemp("", "galaxy.config.*.toml")
	if err != nil {
		return diagnostics
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(content)); err != nil {
		return diagnostics
	}
	tmpFile.Close()

	cfg, err := config.Load(tmpFile.Name())
	if err != nil {
		errorMsg := err.Error()
		line := uint32(0)

		if strings.Contains(errorMsg, "line") {
			parts := strings.Split(errorMsg, "line ")
			if len(parts) > 1 {
				var lineNum int
				if _, scanErr := strings.NewReader(parts[1]).Read([]byte{byte(lineNum)}); scanErr == nil {
					line = uint32(lineNum - 1)
				}
			}
		}

		diagnostics = append(diagnostics, protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: line, Character: 0},
				End:   protocol.Position{Line: line, Character: 100},
			},
			Severity: protocol.DiagnosticSeverityError,
			Source:   "galaxy-config",
			Message:  errorMsg,
		})
		return diagnostics
	}

	if cfg != nil {
		diagnostics = append(diagnostics, validateConfigCompatibility(cfg)...)
	}

	return diagnostics
}

func validateConfigCompatibility(cfg *config.Config) []protocol.Diagnostic {
	diagnostics := make([]protocol.Diagnostic, 0)

	if cfg.Adapter.Name == config.AdapterVercel && cfg.Output.Type != config.OutputStatic {
		diagnostics = append(diagnostics, protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 100},
			},
			Severity: protocol.DiagnosticSeverityError,
			Source:   "galaxy-config",
			Message:  "Vercel adapter only supports output.type = \"static\". Use adapter.name = \"standalone\" for SSR.",
		})
	}

	if cfg.Adapter.Name == config.AdapterNetlify && cfg.Output.Type != config.OutputStatic {
		diagnostics = append(diagnostics, protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 100},
			},
			Severity: protocol.DiagnosticSeverityError,
			Source:   "galaxy-config",
			Message:  "Netlify adapter only supports output.type = \"static\". Use adapter.name = \"standalone\" for SSR.",
		})
	}

	if cfg.Adapter.Name == config.AdapterCloudflare && cfg.Output.Type != config.OutputStatic {
		diagnostics = append(diagnostics, protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 100},
			},
			Severity: protocol.DiagnosticSeverityError,
			Source:   "galaxy-config",
			Message:  "Cloudflare adapter only supports output.type = \"static\". Use adapter.name = \"standalone\" for SSR.",
		})
	}

	return diagnostics
}
