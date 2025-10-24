package wasm

import (
	"fmt"
)

func GenerateLoader(wasmPath string) string {
	return fmt.Sprintf(`
(function() {
	const wasmPath = "%s";
	const wasmKey = wasmPath.replace(/\//g, '_');
	
	window.__galaxyWasmModules = window.__galaxyWasmModules || {};
	
	async function loadWasmModule() {
		if (window.__galaxyWasmModules[wasmKey]) {
			console.log('[WASM] Module already loaded:', wasmKey);
			return;
		}

		try {
			const go = new Go();
			const result = await WebAssembly.instantiateStreaming(
				fetch(wasmPath + '?t=' + Date.now()), 
				go.importObject
			);
			
			go.run(result.instance);
			
			window.__galaxyWasmModules[wasmKey] = {
				instance: result.instance,
				go: go,
				cleanup: function() {
					if (window.galaxyWasmCleanup) {
						try {
							window.galaxyWasmCleanup();
						} catch(e) {
							console.warn('[WASM] Cleanup failed:', e);
						}
					}
				}
			};
			
			console.log('[WASM] Module loaded:', wasmKey);
		} catch (err) {
			console.error('[WASM] Failed to load module:', err);
		}
	}
	
	loadWasmModule();
})();
`, wasmPath)
}

func GetWasmExecJS() string {
	return `../misc/wasm/wasm_exec.js`
}
