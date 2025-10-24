//go:build !wasm
// +build !wasm

package wasmdom

import "syscall/js"

type HMRModule struct{}

func NewHMRModule(moduleID string) *HMRModule { return &HMRModule{} }
func (m *HMRModule) Accept(callback func()) {}
func (m *HMRModule) OnDispose(handler func()) {}
func (m *HMRModule) TrackListener(el Element, event string, handler js.Func) {}
func (m *HMRModule) SaveState(key string, value interface{}) {}
func (m *HMRModule) LoadState(key string) js.Value { return js.Undefined() }
