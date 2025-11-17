package lsp

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/withgalaxy/galaxy/pkg/parser"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

type Server struct {
	conn           jsonrpc2.Conn
	cache          map[protocol.DocumentURI]*DocumentState
	cacheMu        sync.RWMutex
	project        *ProjectContext
	rootPath       string
	gopls          *GoplsProxy
	componentCache map[string]*ComponentInfo
	componentMu    sync.RWMutex
}

type DocumentState struct {
	URI     protocol.DocumentURI
	Content string
	Version int32
}

func NewServer(conn jsonrpc2.Conn) *Server {
	return &Server{
		conn:           conn,
		cache:          make(map[protocol.DocumentURI]*DocumentState),
		componentCache: make(map[string]*ComponentInfo),
	}
}

func (s *Server) Initialize(ctx context.Context, params *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	// Extract root path
	if params.RootURI != "" {
		s.rootPath = string(params.RootURI)[7:] // Strip file://
	} else if params.RootPath != "" {
		s.rootPath = params.RootPath
	}

	// Load project context
	if s.rootPath != "" {
		if project, err := NewProjectContext(s.rootPath); err == nil {
			s.project = project
		}

		// Initialize gopls proxy
		if gopls, err := NewGoplsProxy(s.rootPath); err == nil {
			s.gopls = gopls
		}
	}

	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			TextDocumentSync: protocol.TextDocumentSyncOptions{
				OpenClose: true,
				Change:    protocol.TextDocumentSyncKindFull,
				Save:      &protocol.SaveOptions{IncludeText: false},
			},
			CompletionProvider: &protocol.CompletionOptions{
				TriggerCharacters: []string{"{", ":", " ", ".", "=", "["},
			},
			HoverProvider:      true,
			DefinitionProvider: interface{}(true),
		},
		ServerInfo: &protocol.ServerInfo{
			Name:    "gxc-language-server",
			Version: "0.15.0",
		},
	}, nil
}

func (s *Server) Initialized(ctx context.Context, params *protocol.InitializedParams) error {
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return nil
}

func (s *Server) Exit(ctx context.Context) error {
	return nil
}

func (s *Server) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) error {
	s.cacheMu.Lock()
	s.cache[params.TextDocument.URI] = &DocumentState{
		URI:     params.TextDocument.URI,
		Content: params.TextDocument.Text,
		Version: params.TextDocument.Version,
	}
	s.cacheMu.Unlock()

	go s.publishDiagnostics(ctx, params.TextDocument.URI, params.TextDocument.Text)

	return nil
}

func (s *Server) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) error {
	if len(params.ContentChanges) == 0 {
		return nil
	}

	change := params.ContentChanges[0]
	newContent := change.Text

	s.cacheMu.Lock()
	if state, ok := s.cache[params.TextDocument.URI]; ok {
		state.Content = newContent
		state.Version = params.TextDocument.Version
	}
	s.cacheMu.Unlock()

	go s.publishDiagnostics(ctx, params.TextDocument.URI, newContent)

	return nil
}

func (s *Server) DidClose(ctx context.Context, params *protocol.DidCloseTextDocumentParams) error {
	s.cacheMu.Lock()
	delete(s.cache, params.TextDocument.URI)
	s.cacheMu.Unlock()

	return nil
}

func (s *Server) DidSave(ctx context.Context, params *protocol.DidSaveTextDocumentParams) error {
	s.invalidateComponentCache(params.TextDocument.URI)
	return nil
}

func (s *Server) publishDiagnostics(ctx context.Context, uri protocol.DocumentURI, content string) {
	var diagnostics []protocol.Diagnostic

	if IsTOMLFile(uri) {
		diagnostics = s.analyzeTOML(content)
	} else {
		diagnostics = s.analyze(content)
	}

	err := s.conn.Notify(ctx, "textDocument/publishDiagnostics", &protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	})

	if err != nil {
		fmt.Printf("Error publishing diagnostics: %v\n", err)
	}
}

func (s *Server) Completion(ctx context.Context, params *protocol.CompletionParams) (*protocol.CompletionList, error) {
	// Write to stderr so it shows up in logs
	fmt.Fprintf(os.Stderr, "=== COMPLETION REQUEST === URI=%s Line=%d Char=%d\n", params.TextDocument.URI, params.Position.Line, params.Position.Character)

	s.cacheMu.RLock()
	state, ok := s.cache[params.TextDocument.URI]
	s.cacheMu.RUnlock()

	if !ok {
		fmt.Fprintf(os.Stderr, "=== NO CACHED STATE ===\n")
		return &protocol.CompletionList{Items: []protocol.CompletionItem{}}, nil
	}

	// Check if TOML file
	if IsTOMLFile(params.TextDocument.URI) {
		fmt.Fprintf(os.Stderr, "=== TOML FILE - USING TOML COMPLETIONS ===\n")
		return s.getTOMLCompletions(state.Content, params.Position)
	}

	// Check if in frontmatter - delegate to gopls
	if s.gopls != nil && IsInFrontmatter(params.Position, state.Content) {
		fmt.Fprintf(os.Stderr, "=== IN FRONTMATTER - DELEGATING TO GOPLS ===\n")
		return s.getGoplsCompletion(ctx, params.TextDocument.URI, state.Content, params.Position)
	}

	// Check if in script tag - delegate to gopls
	if s.gopls != nil && IsInScript(params.Position, state.Content) {
		fmt.Fprintf(os.Stderr, "=== IN SCRIPT TAG - DELEGATING TO GOPLS ===\n")
		return s.getGoplsScriptCompletion(ctx, params.TextDocument.URI, state.Content, params.Position)
	}

	fmt.Fprintf(os.Stderr, "=== IN TEMPLATE - USING GXC COMPLETIONS (gopls=%v) ===\n", s.gopls != nil)

	// Otherwise use gxc template logic
	items := s.getCompletions(state.Content, params.Position)

	return &protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

func (s *Server) Hover(ctx context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	s.cacheMu.RLock()
	state, ok := s.cache[params.TextDocument.URI]
	s.cacheMu.RUnlock()

	if !ok {
		return nil, nil
	}

	// Check if TOML file
	if IsTOMLFile(params.TextDocument.URI) {
		return s.getTOMLHover(state.Content, params.Position)
	}

	// Check if in frontmatter - delegate to gopls
	if s.gopls != nil && IsInFrontmatter(params.Position, state.Content) {
		return s.getGoplsHover(ctx, params.TextDocument.URI, state.Content, params.Position)
	}

	// Otherwise use gxc logic
	hover := s.getHover(state.Content, params.Position)
	return hover, nil
}

func (s *Server) getGoplsCompletion(ctx context.Context, uri protocol.DocumentURI, content string, pos protocol.Position) (*protocol.CompletionList, error) {
	fmt.Fprintf(os.Stderr, "=== getGoplsCompletion START: uri=%s, pos=%d:%d\n", uri, pos.Line, pos.Character)

	// Parse content to extract frontmatter
	comp, err := parser.Parse(content)
	if err != nil || comp.Frontmatter == "" {
		fmt.Fprintf(os.Stderr, "=== Parse error or empty frontmatter: %v\n", err)
		return &protocol.CompletionList{Items: []protocol.CompletionItem{}}, nil
	}

	fmt.Fprintf(os.Stderr, "=== Frontmatter length: %d\n", len(comp.Frontmatter))

	// Create virtual Go file
	goPath, err := s.gopls.CreateVirtualGoFile(string(uri), comp.Frontmatter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "=== CreateVirtualGoFile error: %v\n", err)
		return &protocol.CompletionList{Items: []protocol.CompletionItem{}}, nil
	}

	fmt.Fprintf(os.Stderr, "=== Created virtual file: %s\n", goPath)

	// Map position
	pm := NewPositionMapper(content)
	goLine, goChar := pm.GxcToGo(int(pos.Line), int(pos.Character))

	fmt.Fprintf(os.Stderr, "=== Mapped position %d:%d -> %d:%d\n", pos.Line, pos.Character, goLine, goChar)

	// Request completion from gopls
	result, err := s.gopls.Completion(ctx, goPath, goLine, goChar)
	if err != nil {
		fmt.Fprintf(os.Stderr, "=== Gopls completion error: %v\n", err)
		return &protocol.CompletionList{Items: []protocol.CompletionItem{}}, nil
	}

	fmt.Fprintf(os.Stderr, "=== Gopls returned %d completions\n", len(result.Items))

	// Log first few completions for debugging
	for i := 0; i < len(result.Items) && i < 5; i++ {
		fmt.Fprintf(os.Stderr, "===   [%d] %s (kind=%v)\n", i, result.Items[i].Label, result.Items[i].Kind)
	}

	// CRITICAL: Transform all completion item positions from .go back to .gxc
	// Without this, TextEdit ranges will reference wrong line numbers
	for i := range result.Items {
		result.Items[i] = pm.TransformCompletionItem(result.Items[i])
	}

	fmt.Fprintf(os.Stderr, "=== Transformed completion positions from .go to .gxc\n")

	return result, nil
}

func (s *Server) getGoplsHover(ctx context.Context, uri protocol.DocumentURI, content string, pos protocol.Position) (*protocol.Hover, error) {
	// Parse content to extract frontmatter
	comp, err := parser.Parse(content)
	if err != nil || comp.Frontmatter == "" {
		return nil, nil
	}

	// Create virtual Go file
	goPath, err := s.gopls.CreateVirtualGoFile(string(uri), comp.Frontmatter)
	if err != nil {
		return nil, nil
	}

	// Map position
	pm := NewPositionMapper(content)
	goLine, goChar := pm.GxcToGo(int(pos.Line), int(pos.Character))

	// Request hover from gopls
	return s.gopls.Hover(ctx, goPath, goLine, goChar)
}

func (s *Server) getGoplsScriptCompletion(ctx context.Context, uri protocol.DocumentURI, content string, pos protocol.Position) (*protocol.CompletionList, error) {
	fmt.Fprintf(os.Stderr, "=== getGoplsScriptCompletion START: uri=%s, pos=%d:%d\n", uri, pos.Line, pos.Character)

	// Find script at cursor position
	scriptContent, scriptStart, _, found := FindScriptAtPosition(content, pos)
	if !found {
		fmt.Fprintf(os.Stderr, "=== No script found at position\n")
		return &protocol.CompletionList{Items: []protocol.CompletionItem{}}, nil
	}

	fmt.Fprintf(os.Stderr, "=== Script content length: %d, starts at line %d\n", len(scriptContent), scriptStart)

	// Create virtual Go file from script
	goPath, err := s.gopls.CreateVirtualGoFileFromScript(string(uri), scriptContent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "=== CreateVirtualGoFileFromScript error: %v\n", err)
		return &protocol.CompletionList{Items: []protocol.CompletionItem{}}, nil
	}

	fmt.Fprintf(os.Stderr, "=== Created virtual script file: %s\n", goPath)

	// Map position
	spm := NewScriptPositionMapper(scriptContent, scriptStart)
	goLine, goChar := spm.GxcToGo(int(pos.Line), int(pos.Character))

	fmt.Fprintf(os.Stderr, "=== Mapped script position %d:%d -> %d:%d\n", pos.Line, pos.Character, goLine, goChar)

	// Request completion from gopls
	result, err := s.gopls.Completion(ctx, goPath, goLine, goChar)
	if err != nil {
		fmt.Fprintf(os.Stderr, "=== Gopls script completion error: %v\n", err)
		return &protocol.CompletionList{Items: []protocol.CompletionItem{}}, nil
	}

	fmt.Fprintf(os.Stderr, "=== Gopls returned %d script completions\n", len(result.Items))

	// Log first few completions for debugging
	for i := 0; i < len(result.Items) && i < 5; i++ {
		fmt.Fprintf(os.Stderr, "===   [%d] %s (kind=%v)\n", i, result.Items[i].Label, result.Items[i].Kind)
	}

	// Transform all completion item positions from .go back to .gxc
	for i := range result.Items {
		result.Items[i] = spm.TransformCompletionItem(result.Items[i])
	}

	fmt.Fprintf(os.Stderr, "=== Transformed script completion positions from .go to .gxc\n")

	return result, nil
}

func (s *Server) Definition(ctx context.Context, params *protocol.DefinitionParams) ([]protocol.Location, error) {
	s.cacheMu.RLock()
	state, ok := s.cache[params.TextDocument.URI]
	s.cacheMu.RUnlock()

	if !ok {
		return nil, nil
	}

	// Detect what cursor is on
	target := detectDefinitionContext(state.Content, params.Position)

	switch target.Context {
	case ContextComponentName:
		return s.goToComponent(target.ComponentName)

	case ContextPropName:
		return s.goToPropDefinition(target.ComponentName, target.PropName)

	case ContextVariableName:
		return s.goToVariableDefinition(params.TextDocument.URI, state.Content, target.VariableName)

	default:
		return nil, nil
	}
}

func (s *Server) loadComponentInfo(componentPath string) (*ComponentInfo, error) {
	s.componentMu.RLock()
	cached, ok := s.componentCache[componentPath]
	s.componentMu.RUnlock()

	if ok {
		return cached, nil
	}

	content, err := os.ReadFile(componentPath)
	if err != nil {
		return nil, err
	}

	info, err := ParseComponentProps(componentPath, string(content))
	if err != nil {
		return nil, err
	}

	s.componentMu.Lock()
	s.componentCache[componentPath] = info
	s.componentMu.Unlock()

	return info, nil
}

func (s *Server) invalidateComponentCache(uri protocol.DocumentURI) {
	path := string(uri)
	if strings.HasPrefix(path, "file://") {
		path = path[7:]
	}

	s.componentMu.Lock()
	delete(s.componentCache, path)
	s.componentMu.Unlock()

	// Re-analyze all open files that might use this component
	s.reanalyzeOpenFiles(context.Background())
}

func (s *Server) reanalyzeOpenFiles(ctx context.Context) {
	s.cacheMu.RLock()
	openFiles := make(map[protocol.DocumentURI]*DocumentState)
	for uri, state := range s.cache {
		openFiles[uri] = state
	}
	s.cacheMu.RUnlock()

	for uri, state := range openFiles {
		go s.publishDiagnostics(ctx, uri, state.Content)
	}
}
