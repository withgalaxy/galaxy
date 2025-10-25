# HMR Future Enhancements

## Current Status

✅ **Working:**
- Style hot swap (instant, no reload)
- Template morphing (DOM updates without reload)
- WASM hot reload with state preservation (opt-in via hmr* helpers)
- Auto-injected HMR helpers (no imports needed)

## Planned Enhancements

### 1. Documentation & Examples

**Priority: High**

- [ ] Add HMR usage guide to docs
- [ ] Create comprehensive counter example with HMR
- [ ] Best practices guide
- [ ] Common patterns (state preservation, cleanup)
- [ ] Troubleshooting guide

**Example topics:**
- How to use hmrAccept/hmrOnDispose
- State preservation patterns
- Event listener management
- When HMR triggers vs full reload

### 2. Error Overlay & UX Polish

**Priority: High - Huge DX Win**

- [ ] Error overlay for compile failures (Vite-style)
  - Show compilation errors without losing page state
  - Syntax highlighting
  - Click to open in editor
  - Auto-dismiss on fix
- [ ] Better console messages
  - Color-coded HMR logs
  - Change type indicators
  - Performance metrics
- [ ] Visual feedback
  - Toast notifications for updates
  - Progress indicator during WASM compile
  - Success/error animations
- [ ] Browser notifications (optional)

### 3. Component-Level HMR

**Priority: Medium**

- [ ] Auto-detect component changes
- [ ] Re-render only affected components
- [ ] Component state preservation
- [ ] Slot content preservation
- [ ] Props diffing

**Approach:**
- Track component instances in browser
- Re-compile only changed component
- Patch component DOM in-place
- Preserve component-scoped state

### 4. Middleware Hot Reload

**Priority: Medium**

- [ ] Detect middleware.go changes
- [ ] Recompile middleware on-the-fly
- [ ] Reload middleware chain without restart
- [ ] Test request flow after reload

**Current:** Middleware changes trigger server restart
**Goal:** Hot reload middleware like routes

### 5. Performance Optimizations

**Priority: Medium**

- [ ] Debounce rapid file changes
- [ ] Lazy load morphdom (only when needed)
- [ ] Skip unchanged routes in batch updates
- [ ] Cache compiled modules more aggressively
- [ ] Parallel WASM compilation
- [ ] Incremental template rendering

**Metrics to track:**
- Time to hot reload
- Memory usage
- WebSocket message size
- Cache hit rate

### 6. Advanced WASM Features

**Priority: Low**

- [ ] Auto-detect state variables (analyze AST)
- [ ] Auto-generate hmrAccept boilerplate
- [ ] WASM module preloading
- [ ] Shared state across modules
- [ ] Hot reload WASM imports

**Smart HMR:**
```go
// Auto-detect this pattern and inject HMR
var count int // <- automatically preserved
```

### 7. Production Safety

**Priority: High - Before Release**

- [ ] Ensure HMR fully disabled in production builds
- [ ] Tree-shake all HMR code from bundles
- [ ] Zero runtime overhead when disabled
- [ ] Build-time dead code elimination
- [ ] Verify no HMR leaks in dist/

**Check:**
- No `__hmr` endpoints in prod
- No HMR client script
- No WebSocket connections
- Bundle size unchanged

### 8. Multi-Page HMR

**Priority: Low**

- [ ] Update all open tabs simultaneously
- [ ] Broadcast across browser windows
- [ ] Sync state between tabs
- [ ] Multi-device sync (optional)

### 9. Testing & Reliability

**Priority: Medium**

- [ ] HMR integration tests
- [ ] E2E tests for each HMR type
- [ ] Stress test (rapid changes)
- [ ] Memory leak detection
- [ ] Error recovery tests

**Test scenarios:**
- Style → Template → WASM changes in rapid succession
- Invalid syntax recovery
- Network interruption handling
- Large file changes
- Concurrent edits

### 10. Developer Tools

**Priority: Low**

- [ ] HMR debug panel
- [ ] Module dependency graph
- [ ] Hot reload history
- [ ] Performance profiler
- [ ] HMR analytics

**Possible features:**
- Visualize what changed
- Show HMR decision tree
- Benchmark reload times
- Track state mutations

## Implementation Priority

**Phase 1 (Next):**
1. Error overlay
2. Better console messages
3. Documentation

**Phase 2 (Soon):**
1. Component-level HMR
2. Performance optimizations
3. Production safety audit

**Phase 3 (Later):**
1. Middleware hot reload
2. Advanced WASM features
3. Developer tools

## Quick Wins

These can be implemented quickly with high impact:

1. **Error overlay** - 2-3 hours, massive DX improvement
2. **Toast notifications** - 1 hour, better feedback
3. **Debounce changes** - 30 min, prevents thrashing
4. **Better logs** - 1 hour, easier debugging
5. **Lazy morphdom** - 30 min, faster initial load

## Known Limitations

- WASM HMR requires manual hmrAccept() opt-in
- Template morphing may fail with complex dynamic content
- Event listeners need manual tracking for cleanup
- Go WASM runtime threading model limits some patterns
- No source maps for WASM yet

## Questions to Explore

- Can we auto-detect HMR acceptance from code patterns?
- Should we support partial WASM updates (function-level)?
- How to handle breaking changes in hot reload?
- What's the right balance between auto-magic and explicit control?
- Should HMR work with SSG (static generation)?

## Resources

- Vite HMR API: https://vitejs.dev/guide/api-hmr.html
- React Fast Refresh: https://github.com/facebook/react/tree/main/packages/react-refresh
- Morphdom: https://github.com/patrick-steele-idem/morphdom
- Go WASM: https://github.com/golang/go/wiki/WebAssembly

---

**Last Updated:** 2025-10-24
**Status:** Phase 0 Complete (Basic HMR working)
