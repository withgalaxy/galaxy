package wasm

import (
	"fmt"
)

func GenerateLoader(wasmPath string) string {
	return fmt.Sprintf(`
(function() {
	const wasmPath = "%s";
	const moduleId = wasmPath.replace(/[^a-zA-Z0-9]/g, '_');
	
	window.__galaxyWasmModules = window.__galaxyWasmModules || {};
	window.__galaxyWasmAcceptHandlers = window.__galaxyWasmAcceptHandlers || {};
	window.__galaxyWasmState = window.__galaxyWasmState || {};
	
	window.loadWasmModule = async function(modId, path, hash, isHotUpdate = false) {
		const oldModule = window.__galaxyWasmModules[modId];
		
		if (isHotUpdate && oldModule) {
			console.log('[WASM HMR] Disposing old module:', modId);
			
			if (oldModule.disposeHandler) {
				try {
					await oldModule.disposeHandler();
				} catch (e) {
					console.warn('[WASM HMR] Dispose failed:', e);
				}
			}
			
			if (oldModule.listeners) {
				oldModule.listeners.forEach(({ el, event, handler }) => {
					try {
						el.removeEventListener(event, handler);
					} catch (e) {
						console.warn('[WASM HMR] Listener cleanup failed:', e);
					}
				});
			}
		}
		
		try {
			const go = new Go();
			const cacheBuster = hash || Date.now();
			const result = await WebAssembly.instantiateStreaming(
				fetch(path + '?t=' + cacheBuster),
				go.importObject
			);
			
			window.__galaxyWasmModules[modId] = {
				instance: result.instance,
				go: go,
				listeners: [],
				disposeHandler: null
			};
			
			go.run(result.instance);
			
			if (isHotUpdate && window.__galaxyWasmAcceptHandlers[modId]) {
				console.log('[WASM HMR] Calling accept handler for:', modId);
				await window.__galaxyWasmAcceptHandlers[modId]();
			}
			
			console.log('[WASM] Module loaded:', modId);
		} catch (err) {
			console.error('[WASM] Failed to load module:', err);
			throw err;
		}
	};
	
	loadWasmModule(moduleId, wasmPath, null, false);
})();
`, wasmPath)
}

func GetWasmExecJS() string {
	return `../misc/wasm/wasm_exec.js`
}
