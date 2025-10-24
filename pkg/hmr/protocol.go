package hmr

type MessageType string

const (
	MsgTypeConnect        MessageType = "connect"
	MsgTypeReload         MessageType = "reload"
	MsgTypeStyleUpdate    MessageType = "style-update"
	MsgTypeScriptReload   MessageType = "script-reload"
	MsgTypeTemplateUpdate MessageType = "template-update"
	MsgTypeWasmReload     MessageType = "wasm-reload"
)

type Message struct {
	Type     MessageType            `json:"type"`
	Path     string                 `json:"path,omitempty"`
	Content  string                 `json:"content,omitempty"`
	Hash     string                 `json:"hash,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}
