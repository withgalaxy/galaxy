//go:build js && wasm
// +build js,wasm

package wasmdom

import (
	"syscall/js"
)

type HMRModule struct {
	moduleID string
	cleanup  []js.Func
}

func NewHMRModule(moduleID string) *HMRModule {
	return &HMRModule{
		moduleID: moduleID,
		cleanup:  make([]js.Func, 0),
	}
}

func (m *HMRModule) Accept(callback func()) {
	ensureGlobals()

	handlers := js.Global().Get("__galaxyWasmAcceptHandlers")
	cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		callback()
		return nil
	})
	handlers.Set(m.moduleID, cb)
}

func (m *HMRModule) OnDispose(handler func()) {
	ensureGlobals()

	modules := js.Global().Get("__galaxyWasmModules")
	module := modules.Get(m.moduleID)

	if !module.IsUndefined() {
		cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			for _, fn := range m.cleanup {
				fn.Release()
			}
			handler()
			return nil
		})
		module.Set("disposeHandler", cb)
	}
}

func (m *HMRModule) TrackListener(el Element, event string, handler js.Func) {
	m.cleanup = append(m.cleanup, handler)

	ensureGlobals()
	modules := js.Global().Get("__galaxyWasmModules")
	module := modules.Get(m.moduleID)

	if !module.IsUndefined() {
		listeners := module.Get("listeners")
		if listeners.IsUndefined() {
			listeners = js.Global().Get("Array").New()
			module.Set("listeners", listeners)
		}

		listenerObj := js.Global().Get("Object").New()
		listenerObj.Set("el", el.Value)
		listenerObj.Set("event", event)
		listenerObj.Set("handler", handler)

		listeners.Call("push", listenerObj)
	}
}

func (m *HMRModule) SaveState(key string, value interface{}) {
	ensureGlobals()
	state := js.Global().Get("__galaxyWasmState")

	moduleState := state.Get(m.moduleID)
	if moduleState.IsUndefined() {
		moduleState = js.Global().Get("Object").New()
		state.Set(m.moduleID, moduleState)
	}

	moduleState.Set(key, js.ValueOf(value))
}

func (m *HMRModule) LoadState(key string) js.Value {
	ensureGlobals()
	state := js.Global().Get("__galaxyWasmState")
	moduleState := state.Get(m.moduleID)

	if moduleState.IsUndefined() {
		return js.Undefined()
	}

	return moduleState.Get(key)
}

func ensureGlobals() {
	if js.Global().Get("__galaxyWasmModules").IsUndefined() {
		js.Global().Set("__galaxyWasmModules", js.Global().Get("Object").New())
	}
	if js.Global().Get("__galaxyWasmAcceptHandlers").IsUndefined() {
		js.Global().Set("__galaxyWasmAcceptHandlers", js.Global().Get("Object").New())
	}
	if js.Global().Get("__galaxyWasmState").IsUndefined() {
		js.Global().Set("__galaxyWasmState", js.Global().Get("Object").New())
	}
}

// Global helper functions for HMR state management
// These allow scripts to use hmrSaveState/hmrLoadState directly without creating an HMRModule
func HmrSaveState(key string, value interface{}) {
	ensureGlobals()
	state := js.Global().Get("__galaxyWasmState")
	state.Set(key, js.ValueOf(value))
}

func HmrLoadState(key string) js.Value {
	ensureGlobals()
	state := js.Global().Get("__galaxyWasmState")
	return state.Get(key)
}
