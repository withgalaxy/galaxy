(function() {
  if (window.location.hostname !== 'localhost' && window.location.hostname !== '127.0.0.1') {
    return;
  }

  class GalaxyHMR {
    constructor() {
      this.morphdom = null;
      this.loadMorph();
      this.loadOverlay();
      this.connect();
      this.restoreState();
    }

    async loadMorph() {
      try {
        const script = document.createElement('script');
        script.src = '/__hmr/morph.js';
        script.onload = () => {
          this.morphdom = window.morphdom || morphdom;
          this.log('Morphdom loaded', 'info');
        };
        document.head.appendChild(script);
      } catch (e) {
        this.log('Failed to load morphdom', 'error', e);
      }
    }

    async loadOverlay() {
      try {
        const script = document.createElement('script');
        script.src = '/__hmr/overlay.js';
        document.head.appendChild(script);
      } catch (e) {
        this.log('Failed to load overlay', 'error', e);
      }
    }

    log(message, level = 'info', data = null) {
      const prefix = '%c[HMR]%c';
      const prefixStyle = 'color: #61dafb; font-weight: bold;';
      const messageStyle = 'color: inherit;';
      
      const styles = {
        info: 'color: #3498db;',
        success: 'color: #27ae60;',
        warn: 'color: #f39c12;',
        error: 'color: #e74c3c;'
      };

      const fullMessage = `${prefix} ${message}`;
      
      if (level === 'error' && data) {
        console.error(fullMessage, prefixStyle, styles[level], data);
      } else if (level === 'warn') {
        console.warn(fullMessage, prefixStyle, styles[level]);
      } else {
        console.log(fullMessage, prefixStyle, styles[level]);
      }
    }

    connect() {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      this.ws = new WebSocket(`${protocol}//${window.location.host}/__hmr`);
      
      this.ws.onopen = () => {
        this.log('Connected', 'success');
        if (window.__galaxyHmr && window.__galaxyHmr.hideError) {
          window.__galaxyHmr.hideError();
        }
      };

      this.ws.onmessage = (e) => {
        try {
          const msg = JSON.parse(e.data);
          this.handleMessage(msg);
        } catch (err) {
          this.log('Message parse error', 'error', err);
        }
      };

      this.ws.onclose = () => {
        this.log('Disconnected, reconnecting...', 'warn');
        setTimeout(() => this.connect(), 1000);
      };

      this.ws.onerror = (err) => {
        this.log('Connection error', 'error', err);
      };
    }

    async handleMessage(msg) {
      this.log(`Received: ${msg.type}`, 'info');
      
      switch(msg.type) {
        case 'reload':
          this.log('Reloading page...', 'info');
          window.location.reload();
          break;
        
        case 'style-update':
          this.updateStyles(msg.content, msg.hash);
          break;
        
        case 'script-reload':
          this.log('Script changed, reloading...', 'info');
          window.location.reload();
          break;
        
        case 'wasm-reload':
          await this.handleWasmReload(msg);
          break;
        
        case 'template-update':
          await this.handleTemplateUpdate(msg);
          break;
        
        case 'component-update':
          await this.handleComponentUpdate(msg);
          break;
        
        case 'error':
          this.handleError(msg);
          break;
        
        default:
          this.log(`Unknown message type: ${msg.type}`, 'warn');
      }
    }

    handleError(msg) {
      if (window.__galaxyHmr && window.__galaxyHmr.showError) {
        window.__galaxyHmr.showError({
          message: msg.message || 'Build failed',
          stack: msg.stack || ''
        });
      } else {
        this.log(msg.message || 'Build failed', 'error');
      }
    }

    async handleComponentUpdate(msg) {
      if (!this.morphdom) {
        this.log('Morphdom not ready, reloading page...', 'warn');
        this.saveState();
        window.location.reload();
        return;
      }

      try {
        const componentName = msg.metadata?.componentName || 'Component';
        this.log(`Component ${componentName} updated, refreshing page...`, 'info');
        
        const response = await fetch(window.location.href);
        const html = await response.text();
        
        const parser = new DOMParser();
        const newDoc = parser.parseFromString(html, 'text/html');
        const newMain = newDoc.querySelector('main') || newDoc.body;
        const currentMain = document.querySelector('main') || document.body;
        
        this.morphdom(currentMain, newMain.outerHTML);
        
        if (window.__galaxyHmr && window.__galaxyHmr.hideError) {
          window.__galaxyHmr.hideError();
        }
        
        this.log(`Component ${componentName} updated ✨`, 'success');
        this.showToast(`${componentName} updated`);
      } catch (e) {
        this.log('Component update failed', 'error', e);
        this.saveState();
        window.location.reload();
      }
    }

    async handleTemplateUpdate(msg) {
      if (!this.morphdom) {
        this.log('Morphdom not ready, reloading...', 'warn');
        this.saveState();
        window.location.reload();
        return;
      }

      try {
        this.log('Fetching updated template...', 'info');
        const response = await fetch(`/__hmr/render?path=${encodeURIComponent(msg.path)}`);
        const data = await response.json();
        
        if (data.error) {
          this.handleError({ message: data.error, stack: data.stack });
          return;
        }
        
        const container = document.querySelector('main') || document.body;
        this.morphdom(container, `<main>${data.html}</main>`);
        
        if (window.__galaxyHmr && window.__galaxyHmr.hideError) {
          window.__galaxyHmr.hideError();
        }
        
        this.log('Template updated ✨', 'success');
        this.showToast('Template updated');
      } catch (e) {
        this.log('Template update failed', 'error', e);
        this.saveState();
        window.location.reload();
      }
    }

    async handleWasmReload(msg) {
      const moduleId = msg.moduleId || msg.path;
      
      if (window.__galaxyWasmAcceptHandlers && window.__galaxyWasmAcceptHandlers[moduleId]) {
        this.log(`Hot reloading WASM module: ${moduleId}`, 'info');
        try {
          await window.loadWasmModule(moduleId, msg.path, msg.hash, true);
          
          if (window.__galaxyHmr && window.__galaxyHmr.hideError) {
            window.__galaxyHmr.hideError();
          }
          
          this.log('WASM module reloaded ✨', 'success');
          this.showToast('WASM updated');
        } catch (e) {
          this.log('WASM reload failed', 'error', e);
          window.location.reload();
        }
      } else {
        this.log('WASM module cannot hot reload, full reload', 'info');
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
      
      if (window.__galaxyHmr && window.__galaxyHmr.hideError) {
        window.__galaxyHmr.hideError();
      }
      
      this.log('Styles updated ✨', 'success');
      this.showToast('Styles updated');
    }

    showToast(message, type = 'success') {
      if (window.__galaxyHmr && window.__galaxyHmr.showToast) {
        window.__galaxyHmr.showToast(message, type);
      }
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
        
        this.log('State restored', 'info');
      } catch (e) {
        this.log('Failed to restore state', 'error', e);
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
