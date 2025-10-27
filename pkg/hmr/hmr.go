package hmr

import (
	orbit "github.com/cameron-webmatter/orbit/hmr"
)

type Server = orbit.Server
type Message = orbit.Message
type MessageType = orbit.MessageType
type ComponentTracker = orbit.ComponentTracker
type ComponentUsage = orbit.ComponentUsage
type FileTracker = orbit.FileTracker

const (
	MsgTypeConnect         = orbit.MsgTypeConnect
	MsgTypeReload          = orbit.MsgTypeReload
	MsgTypeStyleUpdate     = orbit.MsgTypeStyleUpdate
	MsgTypeScriptReload    = orbit.MsgTypeScriptReload
	MsgTypeTemplateUpdate  = orbit.MsgTypeTemplateUpdate
	MsgTypeWasmReload      = orbit.MsgTypeWasmReload
	MsgTypeError           = orbit.MsgTypeError
	MsgTypeComponentUpdate = orbit.MsgTypeComponentUpdate
)

func NewServer() *Server {
	return orbit.NewServer()
}

func NewComponentTracker() *ComponentTracker {
	return orbit.NewComponentTracker()
}

func NewFileTracker() *FileTracker {
	return orbit.NewFileTracker()
}
