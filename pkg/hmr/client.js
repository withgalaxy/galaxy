(function() {
  if (window.location.hostname !== 'localhost' && window.location.hostname !== '127.0.0.1') {
    return;
  }

  class GalaxyHMR {
    constructor() {
      this.morphdom = null;
      this.loadMorph();
      this.connect();
      this.restoreState();
    }

    async loadMorph() {
      try {
        const script = document.createElement('script');
        script.src = '/__hmr/morph.js';
        script.onload = () => {
          this.morphdom = window.morphdom || morphdom;
          console.log('[HMR] Morphdom loaded');
        };
        document.head.appendChild(script);
      } catch (e) {
        console.warn('[HMR] Failed to load morphdom:', e);
      }
    }

    connect() {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      this.ws = new WebSocket(`${protocol}//${window.location.host}/__hmr`);
      
      this.ws.onopen = () => {
        console.log('[HMR] Connected');
      };

      this.ws.onmessage = (e) => {
        try {
          const msg = JSON.parse(e.data);
          this.handleMessage(msg);
        } catch (err) {
          console.error('[HMR] Message parse error:', err);
        }
      };

      this.ws.onclose = () => {
        console.log('[HMR] Disconnected, reconnecting...');
        setTimeout(() => this.connect(), 1000);
      };

      this.ws.onerror = (err) => {
        console.error('[HMR] Error:', err);
      };
    }

    async handleMessage(msg) {
      console.log('[HMR] Received:', msg.type);
      
      switch(msg.type) {
        case 'reload':
          console.log('[HMR] Reloading page...');
          window.location.reload();
          break;
        
        case 'style-update':
          this.updateStyles(msg.content, msg.hash);
          break;
        
        case 'script-reload':
          console.log('[HMR] Script changed, reloading...');
          window.location.reload();
          break;
        
        case 'wasm-reload':
          await this.handleWasmReload(msg);
          break;
        
        case 'template-update':
          await this.handleTemplateUpdate(msg);
          break;
        
        default:
          console.warn('[HMR] Unknown message type:', msg.type);
      }
    }

    async handleTemplateUpdate(msg) {
      if (!this.morphdom) {
        console.log('[HMR] Morphdom not ready, reloading...');
        this.saveState();
        window.location.reload();
        return;
      }

      try {
        console.log('[HMR] Fetching updated template...');
        const response = await fetch(`/__hmr/render?path=${encodeURIComponent(msg.path)}`);
        const data = await response.json();
        
        const container = document.querySelector('main') || document.body;
        this.morphdom(container, `<main>${data.html}</main>`);
        
        console.log('[HMR] Template updated without reload ✨');
      } catch (e) {
        console.error('[HMR] Template update failed:', e);
        this.saveState();
        window.location.reload();
      }
    }

    async handleWasmReload(msg) {
      const moduleId = msg.moduleId || msg.path;
      
      if (window.__galaxyWasmAcceptHandlers && window.__galaxyWasmAcceptHandlers[moduleId]) {
        console.log('[WASM HMR] Hot reloading module:', moduleId);
        try {
          await window.loadWasmModule(moduleId, msg.path, msg.hash, true);
          console.log('[WASM HMR] Module reloaded ✨');
        } catch (e) {
          console.error('[WASM HMR] Reload failed:', e);
          window.location.reload();
        }
      } else {
        console.log('[WASM HMR] Module cannot hot reload, full reload');
        window.location.reload();
      }
    }

    updateStyles(css, hash) {
      const styleId = `hmr-style-${hash}`;
      let styleEl = document.getElementById(styleId);
      
      if (!styleEl) {
        const oldStyles = document.querySelectorAll('style[id^="hmr-style-"]');
        oldStyles.forEach(s => s.remove());
        
        styleEl = document.createElement('style');
        styleEl.id = styleId;
        document.head.appendChild(styleEl);
      }
      
      styleEl.textContent = css;
      console.log('[HMR] Styles updated without reload ✨');
    }

    saveState() {
      const state = {
        scroll: { x: window.scrollX, y: window.scrollY },
        forms: this.captureFormState()
      };
      sessionStorage.setItem('__hmr_state', JSON.stringify(state));
    }

    captureFormState() {
      const state = {};
      document.querySelectorAll('input, textarea, select').forEach((el, i) => {
        const key = el.id || el.name || `__el_${i}`;
        if (el.type === 'checkbox' || el.type === 'radio') {
          state[key] = el.checked;
        } else {
          state[key] = el.value;
        }
      });
      return state;
    }

    restoreState() {
      const savedState = sessionStorage.getItem('__hmr_state');
      if (!savedState) return;
      
      try {
        const state = JSON.parse(savedState);
        sessionStorage.removeItem('__hmr_state');
        
        if (state.scroll) {
          setTimeout(() => {
            window.scrollTo(state.scroll.x, state.scroll.y);
          }, 0);
        }
        
        if (state.forms) {
          setTimeout(() => {
            document.querySelectorAll('input, textarea, select').forEach((el, i) => {
              const key = el.id || el.name || `__el_${i}`;
              const value = state.forms[key];
              if (value !== undefined) {
                if (el.type === 'checkbox' || el.type === 'radio') {
                  el.checked = value;
                } else {
                  el.value = value;
                }
              }
            });
          }, 0);
        }
        
        console.log('[HMR] State restored');
      } catch (e) {
        console.error('[HMR] Failed to restore state:', e);
      }
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
      new GalaxyHMR();
    });
  } else {
    new GalaxyHMR();
  }
})();
