//go:build js && wasm
// +build js,wasm

package store

import (
	"encoding/json"
	"syscall/js"
)

func saveStateToHMR[T any](key string, value T) {
	ensureHMRGlobals()
	state := js.Global().Get("__galaxyWasmState")

	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return
	}

	state.Set(key, string(jsonBytes))
}

func loadStateFromHMR[T any](key string) (T, bool) {
	var zero T
	ensureHMRGlobals()
	state := js.Global().Get("__galaxyWasmState")
	val := state.Get(key)

	if val.IsUndefined() || val.IsNull() {
		return zero, false
	}

	jsonStr := val.String()
	var result T
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return zero, false
	}

	return result, true
}

func ensureHMRGlobals() {
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
