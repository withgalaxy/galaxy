# Store Package

Reactive state management for Galaxy Go/WASM applications, inspired by Astro's nano stores.

## Features

- **Type-safe** reactive stores using Go generics
- **Three store types**: Atom, Map, Computed
- **HMR integration** - state persists across hot reloads
- **Subscription-based** reactivity
- **Thread-safe** with mutex protection
- **Zero dependencies** - uses only stdlib and syscall/js

## Quick Start

```go
import "github.com/withgalaxy/galaxy/pkg/store"

count := store.NewAtom(0)
doubled := store.NewComputed(count, func(v int) int { return v * 2 })

count.Subscribe(func(val int) {
    fmt.Println("Count:", val)
})

count.Set(5) // Prints: Count: 5
```

## Store Types

### Atom[T]
Single value container with type safety.

```go
stringStore := store.NewAtom("hello")
intStore := store.NewAtom(42)
structStore := store.NewAtom(User{Name: "Alice"})
```

### MapStore
Key-value container for structured data.

```go
user := store.NewMap(map[string]any{
    "name": "Alice",
    "age": 30,
})
user.SetKey("age", 31)
```

### Computed[T]
Derived values from other stores.

```go
doubled := store.NewComputed(count, func(v int) int {
    return v * 2
})
```

## HMR Support

Enable HMR to preserve state across hot reloads:

```go
count := store.NewAtomWithHMR(0, "my-counter")
user := store.NewMapWithHMR(initialData, "user-state")
```

## API

### Atom[T]
- `NewAtom[T](initial T) *Atom[T]`
- `NewAtomWithHMR[T](initial T, key string) *Atom[T]`
- `Get() T`
- `Set(value T)`
- `Update(fn func(T) T)`
- `Subscribe(callback func(T)) Unsubscriber`
- `Value() T` - alias for Get()

### MapStore
- `NewMap(initial map[string]any) *MapStore`
- `NewMapWithHMR(initial map[string]any, key string) *MapStore`
- `Get() map[string]any`
- `GetKey(key string) any`
- `Set(value map[string]any)`
- `SetKey(key string, value any)`
- `DeleteKey(key string)`
- `Subscribe(callback func(map[string]any)) Unsubscriber`

### Computed[T]
- `NewComputed[S, T](source ReadableStore[S], fn func(S) T) *Computed[T]`
- `Get() T`
- `Subscribe(callback func(T)) Unsubscriber`
- `Destroy()` - cleanup source subscription

## Examples

See `galaxy/examples/basic/src/pages/`:
- `store-counter.gxc` - Basic usage
- `store-todos.gxc` - Todo app with complex state

## Documentation

Full documentation: `galaxy-docs/src/content/docs/stores-guide.md`

## Testing

Tests are in `store_test.go` with WASM build tags. Cannot run directly without browser environment.

## Future Extensions

Designed for multi-language support via `ReadableStore` interface abstraction.
