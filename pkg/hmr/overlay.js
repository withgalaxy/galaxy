(function() {
  const overlayId = '__galaxy-hmr-error-overlay';
  const toastId = '__galaxy-hmr-toast';
  
  window.__galaxyHmr = window.__galaxyHmr || {};

  window.__galaxyHmr.showError = function(error) {
    hideToast();
    
    let overlay = document.getElementById(overlayId);
    if (!overlay) {
      overlay = document.createElement('div');
      overlay.id = overlayId;
      document.body.appendChild(overlay);
    }

    const stackLines = (error.stack || '').split('\n').slice(0, 15);
    const stackHtml = stackLines.map(line => {
      line = escapeHtml(line);
      if (line.includes('.gxc:') || line.includes('.go:')) {
        return `<div class="stack-line highlight">${line}</div>`;
      }
      return `<div class="stack-line">${line}</div>`;
    }).join('');

    overlay.innerHTML = `
      <style>
        #${overlayId} {
          position: fixed;
          top: 0;
          left: 0;
          width: 100%;
          height: 100%;
          z-index: 999999;
          background: rgba(0, 0, 0, 0.85);
          backdrop-filter: blur(4px);
          color: #fff;
          font-family: 'SF Mono', Monaco, 'Cascadia Code', 'Roboto Mono', Consolas, 'Courier New', monospace;
          font-size: 14px;
          line-height: 1.5;
          overflow: auto;
          padding: 20px;
          box-sizing: border-box;
        }
        #${overlayId} .container {
          max-width: 1000px;
          margin: 0 auto;
          background: #1a1a1a;
          border-radius: 8px;
          border: 1px solid #333;
          box-shadow: 0 20px 50px rgba(0, 0, 0, 0.5);
        }
        #${overlayId} .header {
          background: linear-gradient(135deg, #e74c3c 0%, #c0392b 100%);
          padding: 20px 24px;
          border-radius: 8px 8px 0 0;
          display: flex;
          align-items: center;
          justify-content: space-between;
        }
        #${overlayId} .header-content {
          display: flex;
          align-items: center;
          gap: 12px;
        }
        #${overlayId} .icon {
          font-size: 24px;
        }
        #${overlayId} .title {
          font-size: 18px;
          font-weight: 600;
          margin: 0;
        }
        #${overlayId} .close-btn {
          background: rgba(255, 255, 255, 0.15);
          border: none;
          color: white;
          width: 32px;
          height: 32px;
          border-radius: 4px;
          cursor: pointer;
          font-size: 20px;
          display: flex;
          align-items: center;
          justify-content: center;
          transition: background 0.2s;
        }
        #${overlayId} .close-btn:hover {
          background: rgba(255, 255, 255, 0.25);
        }
        #${overlayId} .content {
          padding: 24px;
        }
        #${overlayId} .message {
          background: #2a2a2a;
          padding: 16px;
          border-radius: 6px;
          margin-bottom: 20px;
          border-left: 3px solid #e74c3c;
          color: #ff6b6b;
          font-weight: 500;
        }
        #${overlayId} .stack {
          background: #0d0d0d;
          padding: 16px;
          border-radius: 6px;
          overflow-x: auto;
          border: 1px solid #2a2a2a;
        }
        #${overlayId} .stack-title {
          color: #888;
          font-size: 12px;
          text-transform: uppercase;
          letter-spacing: 0.5px;
          margin-bottom: 12px;
          font-weight: 600;
        }
        #${overlayId} .stack-line {
          color: #999;
          margin: 4px 0;
          white-space: pre-wrap;
          word-break: break-all;
        }
        #${overlayId} .stack-line.highlight {
          color: #61dafb;
          font-weight: 500;
          background: rgba(97, 218, 251, 0.1);
          padding: 2px 4px;
          border-radius: 3px;
          margin: 6px 0;
        }
        #${overlayId} .footer {
          padding: 16px 24px;
          background: #0d0d0d;
          border-radius: 0 0 8px 8px;
          color: #666;
          font-size: 12px;
          border-top: 1px solid #2a2a2a;
          display: flex;
          align-items: center;
          gap: 8px;
        }
        #${overlayId} .footer-icon {
          font-size: 14px;
        }
      </style>
      <div class="container">
        <div class="header">
          <div class="header-content">
            <span class="icon">⚠️</span>
            <h1 class="title">Build Failed</h1>
          </div>
          <button class="close-btn" onclick="window.__galaxyHmr.hideError()">✕</button>
        </div>
        <div class="content">
          <div class="message">${escapeHtml(error.message || 'Unknown error')}</div>
          ${stackHtml ? `
            <div class="stack">
              <div class="stack-title">Stack Trace</div>
              ${stackHtml}
            </div>
          ` : ''}
        </div>
        <div class="footer">
          <span class="footer-icon">💡</span>
          <span>Fix the error above and save to trigger hot reload</span>
        </div>
      </div>
    `;

    overlay.style.display = 'block';
  };

  window.__galaxyHmr.hideError = function() {
    const overlay = document.getElementById(overlayId);
    if (overlay) {
      overlay.style.display = 'none';
    }
  };

  window.__galaxyHmr.showToast = function(message, type = 'success') {
    hideToast();
    
    let toast = document.getElementById(toastId);
    if (!toast) {
      toast = document.createElement('div');
      toast.id = toastId;
      document.body.appendChild(toast);
    }

    const icons = {
      success: '✓',
      info: 'ℹ',
      warning: '⚠'
    };

    const colors = {
      success: '#27ae60',
      info: '#3498db',
      warning: '#f39c12'
    };

    toast.innerHTML = `
      <style>
        #${toastId} {
          position: fixed;
          bottom: 24px;
          right: 24px;
          z-index: 999998;
          background: ${colors[type] || colors.success};
          color: white;
          padding: 12px 20px;
          border-radius: 6px;
          font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
          font-size: 14px;
          font-weight: 500;
          box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
          display: flex;
          align-items: center;
          gap: 10px;
          animation: slideIn 0.3s ease-out, fadeOut 0.3s ease-in 2.7s;
          pointer-events: none;
        }
        #${toastId} .icon {
          font-size: 18px;
          font-weight: bold;
        }
        @keyframes slideIn {
          from {
            transform: translateX(400px);
            opacity: 0;
          }
          to {
            transform: translateX(0);
            opacity: 1;
          }
        }
        @keyframes fadeOut {
          from {
            opacity: 1;
          }
          to {
            opacity: 0;
          }
        }
      </style>
      <span class="icon">${icons[type] || icons.success}</span>
      <span>${escapeHtml(message)}</span>
    `;

    toast.style.display = 'flex';
    
    setTimeout(() => {
      hideToast();
    }, 3000);
  };

  function hideToast() {
    const toast = document.getElementById(toastId);
    if (toast) {
      toast.remove();
    }
  }

  function escapeHtml(unsafe) {
    if (typeof unsafe !== 'string') return '';
    return unsafe
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#039;");
  }
})();
