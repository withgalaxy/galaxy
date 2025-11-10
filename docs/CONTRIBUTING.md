# Contributing

Development setup and guidelines.

## Setup

### Clone & Build

```bash
git clone https://github.com/withgalaxy/galaxy
cd galaxy
go mod download
go build -o galaxy ./cmd/galaxy
```

### Install watchexec (optional)

```bash
brew install watchexec  # macOS
cargo install watchexec-cli  # Cross-platform
```

### Development Workflow

```bash
# Terminal 1: Watch & rebuild
make watch

# Terminal 2: Test changes
cd examples/basic
galaxy dev
```

## Project Structure

```
galaxy/
├── cmd/galaxy/          # CLI entry point
├── pkg/
│   ├── cli/            # CLI commands
│   ├── compiler/       # Component compilation
│   ├── router/         # File-based routing
│   ├── build/          # Build system (SSG/SSR/Hybrid)
│   ├── config/         # Configuration
│   ├── plugins/        # Plugin system
│   ├── middleware/     # Middleware
│   ├── endpoints/      # API endpoints
│   ├── template/       # Template engine
│   └── wasmdom/        # WASM DOM API
├── internal/
│   ├── wasm/           # WASM compiler
│   └── assets/         # Asset bundler
├── examples/           # Example projects
└── docs/               # Documentation
```

## Code Guidelines

### Go Style

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt`
- Run `go vet`

### Naming

- Exported: `PascalCase`
- Unexported: `camelCase`

### Error Handling

```go
if err != nil {
    return fmt.Errorf("context: %w", err)
}
```

## Testing

```bash
go test ./...
go test -cover ./...
```

## Commit Messages

Format: `type: description`

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `refactor`: Code refactoring
- `test`: Tests
- `chore`: Build/tooling

Examples:
```
feat: add Hybrid build mode
fix: resolve component cache invalidation
docs: update WASM guide
```

## Pull Requests

1. Fork & branch: `git checkout -b feat/my-feature`
2. Make changes
3. Add tests
4. Run `go test ./...`
5. Commit: `git commit -m "feat: add my feature"`
6. Push: `git push origin feat/my-feature`
7. Open PR

## Makefile Commands

```bash
make install    # Install galaxy CLI
make build      # Build binary
make test       # Run tests
make watch      # Watch & rebuild
make clean      # Clean artifacts
```

---

**See also:** README.md for project overview
