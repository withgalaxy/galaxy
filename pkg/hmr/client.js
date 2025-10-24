(function() {
  if (window.location.hostname !== 'localhost' && window.location.hostname !== '127.0.0.1') {
    return;
  }

  class GalaxyHMR {
    constructor() {
      this.connect();
      this.restoreState();
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

    handleMessage(msg) {
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
          console.log('[HMR] WASM changed, reloading...');
          window.location.reload();
          break;
        
        case 'template-update':
          const state = {
            scroll: { x: window.scrollX, y: window.scrollY },
            forms: this.captureFormState()
          };
          console.log('[HMR] Template changed, reloading with state preservation...');
          sessionStorage.setItem('__hmr_state', JSON.stringify(state));
          window.location.reload();
          break;
        
        default:
          console.warn('[HMR] Unknown message type:', msg.type);
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
