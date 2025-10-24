package lsp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"go.lsp.dev/protocol"
)

// GoplsProxy wraps gopls subprocess
type GoplsProxy struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    *bufio.Reader
	stderr    io.ReadCloser
	reqID     int
	reqMu     sync.Mutex
	rootDir   string
	tempDir   string
	ready     bool
	responses map[int]chan json.RawMessage
	respMu    sync.RWMutex
}

// NewGoplsProxy spawns gopls and initializes it
func NewGoplsProxy(rootDir string) (*GoplsProxy, error) {
	// Create temp directory for virtual Go files
	tempDir := filepath.Join(os.TempDir(), "gxc-gopls")
	os.MkdirAll(tempDir, 0755)

	cmd := exec.Command("gopls", "-mode=stdio")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start gopls: %w", err)
	}

	gp := &GoplsProxy{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    bufio.NewReader(stdout),
		stderr:    stderr,
		rootDir:   rootDir,
		tempDir:   tempDir,
		responses: make(map[int]chan json.RawMessage),
	}

	// Start response reader
	go gp.readResponses()

	// Initialize gopls
	if err := gp.initialize(); err != nil {
		gp.Close()
		return nil, fmt.Errorf("initialize: %w", err)
	}

	return gp, nil
}

func (gp *GoplsProxy) initialize() error {
	initParams := map[string]interface{}{
		"processId": nil,
		"rootUri":   "file://" + gp.rootDir,
		"capabilities": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"completion": map[string]interface{}{
					"completionItem": map[string]interface{}{
						"snippetSupport": true,
					},
				},
				"hover": map[string]interface{}{
					"contentFormat": []string{"markdown"},
				},
			},
		},
	}

	resp, err := gp.request("initialize", initParams)
	if err != nil {
		return err
	}

	// Send initialized notification
	if err := gp.notify("initialized", map[string]interface{}{}); err != nil {
		return err
	}

	gp.ready = true
	_ = resp
	return nil
}

func (gp *GoplsProxy) request(method string, params interface{}) (json.RawMessage, error) {
	gp.reqMu.Lock()
	gp.reqID++
	id := gp.reqID
	gp.reqMu.Unlock()

	// Create response channel
	respChan := make(chan json.RawMessage, 1)
	gp.respMu.Lock()
	gp.responses[id] = respChan
	gp.respMu.Unlock()

	defer func() {
		gp.respMu.Lock()
		delete(gp.responses, id)
		gp.respMu.Unlock()
	}()

	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}

	if err := gp.send(req); err != nil {
		return nil, err
	}

	// Wait for response
	resp := <-respChan
	return resp, nil
}

func (gp *GoplsProxy) notify(method string, params interface{}) error {
	notif := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	return gp.send(notif)
}

func (gp *GoplsProxy) send(msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	content := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(data), data)
	_, err = gp.stdin.Write([]byte(content))
	return err
}

func (gp *GoplsProxy) readResponses() {
	for {
		// Read Content-Length header
		header, err := gp.stdout.ReadString('\n')
		if err != nil {
			return
		}

		header = strings.TrimSpace(header)
		if !strings.HasPrefix(header, "Content-Length:") {
			continue
		}

		var length int
		fmt.Sscanf(header, "Content-Length: %d", &length)

		// Skip empty line
		gp.stdout.ReadString('\n')

		// Read body
		body := make([]byte, length)
		if _, err := io.ReadFull(gp.stdout, body); err != nil {
			return
		}

		// Parse response
		var resp struct {
			ID     *int            `json:"id"`
			Result json.RawMessage `json:"result"`
		}

		if err := json.Unmarshal(body, &resp); err != nil {
			continue
		}

		if resp.ID != nil {
			gp.respMu.RLock()
			if ch, ok := gp.responses[*resp.ID]; ok {
				ch <- resp.Result
			}
			gp.respMu.RUnlock()
		}
	}
}

func (gp *GoplsProxy) Close() error {
	if gp.stdin != nil {
		gp.stdin.Close()
	}
	if gp.cmd != nil && gp.cmd.Process != nil {
		gp.cmd.Process.Kill()
	}
	os.RemoveAll(gp.tempDir)
	return nil
}

// CreateVirtualGoFile converts frontmatter to valid Go file
func (gp *GoplsProxy) CreateVirtualGoFile(uri string, frontmatter string) (string, error) {
	// Extract imports
	imports, code := extractImportsFromFrontmatter(frontmatter)

	// Build valid Go file
	var buf bytes.Buffer
	buf.WriteString("package main\n\n")

	if len(imports) > 0 {
		buf.WriteString("import (\n")
		for _, imp := range imports {
			buf.WriteString("\t" + imp + "\n")
		}
		buf.WriteString(")\n\n")
	}

	buf.WriteString("func _gxcPage() {\n")
	buf.WriteString(code)
	buf.WriteString("\n}\n")

	// Write to temp file
	hash := fmt.Sprintf("%x", uri)[:8]
	goPath := filepath.Join(gp.tempDir, hash+".go")

	goContent := buf.String()
	if err := os.WriteFile(goPath, []byte(goContent), 0644); err != nil {
		return "", err
	}

	// Debug: log the generated Go file
	fmt.Fprintf(os.Stderr, "=== VIRTUAL GO FILE ===\n%s\n=== END ===\n", goContent)

	return goPath, nil
}

func extractImportsFromFrontmatter(code string) ([]string, string) {
	lines := strings.Split(code, "\n")
	var imports []string
	var codeLines []string
	inImportBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "import (") {
			inImportBlock = true
		} else if inImportBlock {
			if strings.Contains(trimmed, ")") {
				inImportBlock = false
			} else if trimmed != "" && !strings.HasPrefix(trimmed, "//") {
				imports = append(imports, trimmed)
			}
		} else if strings.HasPrefix(trimmed, "import ") {
			// Single import
			imp := strings.TrimPrefix(trimmed, "import ")
			imports = append(imports, imp)
		} else {
			codeLines = append(codeLines, line)
		}
	}

	return imports, strings.Join(codeLines, "\n")
}

// CreateVirtualGoFileFromScript converts script content to valid Go file
func (gp *GoplsProxy) CreateVirtualGoFileFromScript(uri string, scriptContent string) (string, error) {
	var buf bytes.Buffer
	buf.WriteString("package main\n\n")

	// Extract imports from script
	lines := strings.Split(scriptContent, "\n")
	var imports []string
	var codeLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "import ") {
			// Extract import path
			imp := strings.TrimPrefix(trimmed, "import ")
			imports = append(imports, imp)
		} else {
			codeLines = append(codeLines, line)
		}
	}

	// Write imports
	if len(imports) > 0 {
		for _, imp := range imports {
			buf.WriteString("import " + imp + "\n")
		}
		buf.WriteString("\n")
	}

	// Wrap code in function
	buf.WriteString("func _gxcScript() {\n")
	code := strings.Join(codeLines, "\n")
	buf.WriteString(code)
	buf.WriteString("\n}\n")

	// Write to temp file with different name to avoid collision with frontmatter
	hash := fmt.Sprintf("%x", uri)[:8]
	goPath := filepath.Join(gp.tempDir, hash+"-script.go")

	goContent := buf.String()
	if err := os.WriteFile(goPath, []byte(goContent), 0644); err != nil {
		return "", err
	}

	// Debug: log the generated Go file
	fmt.Fprintf(os.Stderr, "=== VIRTUAL SCRIPT GO FILE ===\n%s\n=== END ===\n", goContent)

	return goPath, nil
}

// Completion requests completion from gopls
func (gp *GoplsProxy) Completion(ctx context.Context, goPath string, line, char int) (*protocol.CompletionList, error) {
	if !gp.ready {
		return nil, fmt.Errorf("gopls not ready")
	}

	// Open document in gopls
	content, err := os.ReadFile(goPath)
	if err != nil {
		return nil, err
	}

	fileURI := "file://" + goPath

	// Send didOpen
	if err := gp.notify("textDocument/didOpen", map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":        fileURI,
			"languageId": "go",
			"version":    1,
			"text":       string(content),
		},
	}); err != nil {
		return nil, err
	}

	// Request completion
	resp, err := gp.request("textDocument/completion", map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": fileURI,
		},
		"position": map[string]interface{}{
			"line":      line,
			"character": char,
		},
	})
	if err != nil {
		return nil, err
	}

	var result protocol.CompletionList
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Hover requests hover info from gopls
func (gp *GoplsProxy) Hover(ctx context.Context, goPath string, line, char int) (*protocol.Hover, error) {
	if !gp.ready {
		return nil, fmt.Errorf("gopls not ready")
	}

	content, err := os.ReadFile(goPath)
	if err != nil {
		return nil, err
	}

	fileURI := "file://" + goPath

	// Send didOpen
	if err := gp.notify("textDocument/didOpen", map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":        fileURI,
			"languageId": "go",
			"version":    1,
			"text":       string(content),
		},
	}); err != nil {
		return nil, err
	}

	// Request hover
	resp, err := gp.request("textDocument/hover", map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": fileURI,
		},
		"position": map[string]interface{}{
			"line":      line,
			"character": char,
		},
	})
	if err != nil {
		return nil, err
	}

	var result protocol.Hover
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
