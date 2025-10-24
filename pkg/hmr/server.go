package hmr

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Server struct {
	clients   map[*websocket.Conn]bool
	broadcast chan Message
	mu        sync.RWMutex
	upgrader  websocket.Upgrader
}

func NewServer() *Server {
	return &Server{
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan Message, 256),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (s *Server) Start() {
	go s.handleBroadcasts()
}

func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	s.mu.Lock()
	s.clients[conn] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, conn)
		s.mu.Unlock()
		conn.Close()
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (s *Server) handleBroadcasts() {
	for msg := range s.broadcast {
		s.mu.RLock()
		for client := range s.clients {
			err := client.WriteJSON(msg)
			if err != nil {
				client.Close()
				s.mu.RUnlock()
				s.mu.Lock()
				delete(s.clients, client)
				s.mu.Unlock()
				s.mu.RLock()
			}
		}
		s.mu.RUnlock()
	}
}

func (s *Server) BroadcastReload() {
	s.broadcast <- Message{Type: MsgTypeReload}
}

func (s *Server) BroadcastStyleUpdate(path, content, hash string) {
	s.broadcast <- Message{
		Type:    MsgTypeStyleUpdate,
		Path:    path,
		Content: content,
		Hash:    hash,
	}
}

func (s *Server) BroadcastScriptReload(path string) {
	s.broadcast <- Message{
		Type: MsgTypeScriptReload,
		Path: path,
	}
}

func (s *Server) BroadcastWasmReload(path, hash string) {
	s.broadcast <- Message{
		Type: MsgTypeWasmReload,
		Path: path,
		Hash: hash,
	}
}

func (m Message) MarshalJSON() ([]byte, error) {
	type Alias Message
	return json.Marshal(struct{ *Alias }{Alias: (*Alias)(&m)})
}

func (s *Server) BroadcastTemplateUpdate(path string) {
	s.broadcast <- Message{
		Type: MsgTypeTemplateUpdate,
		Path: path,
	}
}
