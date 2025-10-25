package hmr

type MessageType string

const (
	MsgTypeConnect         MessageType = "connect"
	MsgTypeReload          MessageType = "reload"
	MsgTypeStyleUpdate     MessageType = "style-update"
	MsgTypeScriptReload    MessageType = "script-reload"
	MsgTypeTemplateUpdate  MessageType = "template-update"
	MsgTypeWasmReload      MessageType = "wasm-reload"
	MsgTypeError           MessageType = "error"
	MsgTypeComponentUpdate MessageType = "component-update"
)

type Message struct {
	Type     MessageType            `json:"type"`
	Path     string                 `json:"path,omitempty"`
	Content  string                 `json:"content,omitempty"`
	Hash     string                 `json:"hash,omitempty"`
	Message  string                 `json:"message,omitempty"`
	Stack    string                 `json:"stack,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (s *Server) BroadcastWasmReload(path, hash, moduleId string) {
	s.broadcast <- Message{
		Type: MsgTypeWasmReload,
		Path: path,
		Hash: hash,
		Metadata: map[string]interface{}{
			"moduleId": moduleId,
		},
	}
}
