// Minimal morphdom implementation for Galaxy HMR
function morphdom(fromNode, toHTML) {
  const parser = new DOMParser();
  const doc = parser.parseFromString(toHTML, 'text/html');
  const toNode = doc.body;
  
  morphElement(fromNode, toNode);
}

function morphElement(fromEl, toEl) {
  // Sync attributes
  const fromAttrs = fromEl.attributes;
  const toAttrs = toEl.attributes;
  
  for (let i = toAttrs.length - 1; i >= 0; i--) {
    const attr = toAttrs[i];
    if (fromEl.getAttribute(attr.name) !== attr.value) {
      fromEl.setAttribute(attr.name, attr.value);
    }
  }
  
  for (let i = fromAttrs.length - 1; i >= 0; i--) {
    const attr = fromAttrs[i];
    if (!toEl.hasAttribute(attr.name)) {
      fromEl.removeAttribute(attr.name);
    }
  }
  
  // Sync children
  morphChildren(fromEl, toEl);
}

function morphChildren(fromEl, toEl) {
  const fromChildren = fromEl.childNodes;
  const toChildren = toEl.childNodes;
  
  let fromIndex = 0;
  let toIndex = 0;
  
  while (toIndex < toChildren.length) {
    const toChild = toChildren[toIndex];
    const fromChild = fromChildren[fromIndex];
    
    if (!fromChild) {
      fromEl.appendChild(toChild.cloneNode(true));
      toIndex++;
      continue;
    }
    
    if (fromChild.nodeType === Node.TEXT_NODE && toChild.nodeType === Node.TEXT_NODE) {
      if (fromChild.nodeValue !== toChild.nodeValue) {
        fromChild.nodeValue = toChild.nodeValue;
      }
      fromIndex++;
      toIndex++;
      continue;
    }
    
    if (fromChild.nodeType === Node.ELEMENT_NODE && toChild.nodeType === Node.ELEMENT_NODE) {
      if (fromChild.tagName === toChild.tagName) {
        const fromKey = fromChild.getAttribute('data-hmr-key');
        const toKey = toChild.getAttribute('data-hmr-key');
        
        if (fromKey && toKey && fromKey !== toKey) {
          fromEl.replaceChild(toChild.cloneNode(true), fromChild);
        } else {
          morphElement(fromChild, toChild);
        }
        fromIndex++;
        toIndex++;
        continue;
      }
    }
    
    fromEl.replaceChild(toChild.cloneNode(true), fromChild);
    fromIndex++;
    toIndex++;
  }
  
  while (fromIndex < fromChildren.length) {
    fromEl.removeChild(fromChildren[fromIndex]);
  }
}

if (typeof module !== 'undefined' && module.exports) {
  module.exports = morphdom;
}
