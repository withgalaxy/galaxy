package hmr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestNewServer(t *testing.T) {
	srv := NewServer()
	if srv == nil {
		t.Fatal("NewServer returned nil")
	}
	if srv.clients == nil {
		t.Error("clients map not initialized")
	}
	if srv.broadcast == nil {
		t.Error("broadcast channel not initialized")
	}
}

func TestServerWebSocketUpgrade(t *testing.T) {
	srv := NewServer()
	srv.Start()

	server := httptest.NewServer(http.HandlerFunc(srv.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer ws.Close()

	srv.mu.RLock()
	clientCount := len(srv.clients)
	srv.mu.RUnlock()

	if clientCount != 1 {
		t.Errorf("expected 1 client, got %d", clientCount)
	}
}

func TestServerBroadcastReload(t *testing.T) {
	srv := NewServer()
	srv.Start()

	server := httptest.NewServer(http.HandlerFunc(srv.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer ws.Close()

	done := make(chan bool)
	go func() {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			t.Errorf("ReadJSON failed: %v", err)
			return
		}
		if msg.Type != MsgTypeReload {
			t.Errorf("expected MsgTypeReload, got %v", msg.Type)
		}
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)
	srv.BroadcastReload()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Error("timeout waiting for broadcast")
	}
}

func TestServerBroadcastStyleUpdate(t *testing.T) {
	srv := NewServer()
	srv.Start()

	server := httptest.NewServer(http.HandlerFunc(srv.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer ws.Close()

	done := make(chan bool)
	go func() {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			t.Errorf("ReadJSON failed: %v", err)
			return
		}
		if msg.Type != MsgTypeStyleUpdate {
			t.Errorf("expected MsgTypeStyleUpdate, got %v", msg.Type)
		}
		if msg.Path != "/test.css" {
			t.Errorf("expected path /test.css, got %s", msg.Path)
		}
		if msg.Content != "body{color:red}" {
			t.Errorf("expected content body{color:red}, got %s", msg.Content)
		}
		if msg.Hash != "abc123" {
			t.Errorf("expected hash abc123, got %s", msg.Hash)
		}
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)
	srv.BroadcastStyleUpdate("/test.css", "body{color:red}", "abc123")

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Error("timeout waiting for broadcast")
	}
}

func TestServerBroadcastScriptReload(t *testing.T) {
	srv := NewServer()
	srv.Start()

	server := httptest.NewServer(http.HandlerFunc(srv.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer ws.Close()

	done := make(chan bool)
	go func() {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			t.Errorf("ReadJSON failed: %v", err)
			return
		}
		if msg.Type != MsgTypeScriptReload {
			t.Errorf("expected MsgTypeScriptReload, got %v", msg.Type)
		}
		if msg.Path != "/test.js" {
			t.Errorf("expected path /test.js, got %s", msg.Path)
		}
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)
	srv.BroadcastScriptReload("/test.js")

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Error("timeout waiting for broadcast")
	}
}

func TestServerBroadcastTemplateUpdate(t *testing.T) {
	srv := NewServer()
	srv.Start()

	server := httptest.NewServer(http.HandlerFunc(srv.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer ws.Close()

	done := make(chan bool)
	go func() {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			t.Errorf("ReadJSON failed: %v", err)
			return
		}
		if msg.Type != MsgTypeTemplateUpdate {
			t.Errorf("expected MsgTypeTemplateUpdate, got %v", msg.Type)
		}
		if msg.Path != "/test.gxc" {
			t.Errorf("expected path /test.gxc, got %s", msg.Path)
		}
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)
	srv.BroadcastTemplateUpdate("/test.gxc")

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Error("timeout waiting for broadcast")
	}
}

func TestServerBroadcastWasmReload(t *testing.T) {
	srv := NewServer()
	srv.Start()

	server := httptest.NewServer(http.HandlerFunc(srv.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer ws.Close()

	done := make(chan bool)
	go func() {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			t.Errorf("ReadJSON failed: %v", err)
			return
		}
		if msg.Type != MsgTypeWasmReload {
			t.Errorf("expected MsgTypeWasmReload, got %v", msg.Type)
		}
		if msg.Path != "/test.wasm" {
			t.Errorf("expected path /test.wasm, got %s", msg.Path)
		}
		if msg.Hash != "hash123" {
			t.Errorf("expected hash hash123, got %s", msg.Hash)
		}
		if msg.ModuleId != "module1" {
			t.Errorf("expected moduleId module1, got %s", msg.ModuleId)
		}
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)
	srv.BroadcastWasmReload("/test.wasm", "hash123", "module1")

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Error("timeout waiting for broadcast")
	}
}

func TestServerBroadcastError(t *testing.T) {
	srv := NewServer()
	srv.Start()

	server := httptest.NewServer(http.HandlerFunc(srv.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer ws.Close()

	done := make(chan bool)
	go func() {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			t.Errorf("ReadJSON failed: %v", err)
			return
		}
		if msg.Type != MsgTypeError {
			t.Errorf("expected MsgTypeError, got %v", msg.Type)
		}
		if msg.Message != "test error" {
			t.Errorf("expected message 'test error', got %s", msg.Message)
		}
		if msg.Stack != "stack trace" {
			t.Errorf("expected stack 'stack trace', got %s", msg.Stack)
		}
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)
	srv.BroadcastError("test error", "stack trace")

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Error("timeout waiting for broadcast")
	}
}

func TestServerBroadcastComponentUpdate(t *testing.T) {
	srv := NewServer()
	srv.Start()

	server := httptest.NewServer(http.HandlerFunc(srv.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}
	defer ws.Close()

	done := make(chan bool)
	go func() {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			t.Errorf("ReadJSON failed: %v", err)
			return
		}
		if msg.Type != MsgTypeComponentUpdate {
			t.Errorf("expected MsgTypeComponentUpdate, got %v", msg.Type)
		}
		if msg.Path != "/components/Button.gxc" {
			t.Errorf("expected path /components/Button.gxc, got %s", msg.Path)
		}
		if msg.Metadata["componentName"] != "Button" {
			t.Errorf("expected componentName Button, got %v", msg.Metadata["componentName"])
		}
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)
	srv.BroadcastComponentUpdate("/components/Button.gxc", "Button")

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Error("timeout waiting for broadcast")
	}
}

func TestServerMultipleClients(t *testing.T) {
	srv := NewServer()
	srv.Start()

	server := httptest.NewServer(http.HandlerFunc(srv.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	ws1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial 1 failed: %v", err)
	}
	defer ws1.Close()

	ws2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial 2 failed: %v", err)
	}
	defer ws2.Close()

	time.Sleep(50 * time.Millisecond)

	srv.mu.RLock()
	clientCount := len(srv.clients)
	srv.mu.RUnlock()

	if clientCount != 2 {
		t.Errorf("expected 2 clients, got %d", clientCount)
	}

	done := make(chan int, 2)
	receiveMsg := func(ws *websocket.Conn) {
		var msg Message
		if err := ws.ReadJSON(&msg); err == nil && msg.Type == MsgTypeReload {
			done <- 1
		}
	}

	go receiveMsg(ws1)
	go receiveMsg(ws2)

	time.Sleep(50 * time.Millisecond)
	srv.BroadcastReload()

	received := 0
	timeout := time.After(1 * time.Second)
	for received < 2 {
		select {
		case <-done:
			received++
		case <-timeout:
			t.Errorf("timeout: only %d clients received message", received)
			return
		}
	}
}

func TestServerClientDisconnect(t *testing.T) {
	srv := NewServer()
	srv.Start()

	server := httptest.NewServer(http.HandlerFunc(srv.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	srv.mu.RLock()
	clientCount := len(srv.clients)
	srv.mu.RUnlock()

	if clientCount != 1 {
		t.Errorf("expected 1 client, got %d", clientCount)
	}

	ws.Close()
	time.Sleep(100 * time.Millisecond)

	srv.mu.RLock()
	clientCount = len(srv.clients)
	srv.mu.RUnlock()

	if clientCount != 0 {
		t.Errorf("expected 0 clients after disconnect, got %d", clientCount)
	}
}

func TestMessageMarshalJSON(t *testing.T) {
	msg := Message{
		Type:     MsgTypeStyleUpdate,
		Path:     "/test.css",
		Content:  "body{color:red}",
		Hash:     "abc123",
		Message:  "test message",
		Stack:    "stack trace",
		ModuleId: "module1",
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if decoded.Type != msg.Type {
		t.Errorf("Type mismatch: expected %v, got %v", msg.Type, decoded.Type)
	}
	if decoded.Path != msg.Path {
		t.Errorf("Path mismatch: expected %s, got %s", msg.Path, decoded.Path)
	}
	if decoded.Content != msg.Content {
		t.Errorf("Content mismatch: expected %s, got %s", msg.Content, decoded.Content)
	}
}
