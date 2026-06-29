# Contributing to Rivex Go SDK

Thank you for your interest in contributing to Rivex! This guide covers
the Go SDK -- the user-facing framework layer that will link against the
Rust HTTP runtime via CGo FFI.

## Prerequisites

- **Go** 1.22+
- **Rust** toolchain (needed to build `rivex-ffi` when available)
- **golangci-lint** (recommended for linting)

## Repository Layout

```
rivex.go              Main Rivex app (New, Get, Post, Run, etc.)
router.go             Router, Controller interface, route registration
request.go            Context type with all request accessors
response.go           Response types (JSON, HTML, Text, Redirect, Streaming)
middleware.go          Middleware interface + CORS, GZip, RequestID, TrustedHost
depends.go            Dependency injection system
exceptions.go         HTTPException
background.go         BackgroundTask, BackgroundTasks
security.go           APIKeyHeader, HTTPBearer, OAuth2PasswordBearer
uploads.go            UploadFile, multipart parsing
status.go             HTTP status constants
openapi.go            OpenAPI 3.1 generation
testing.go            TestClient
internal/ffi/         CGo bindings to rivex-core (placeholder)
examples/             Example applications
tests/                Test suite
```

## Getting Started

```bash
# Clone the repository
git clone https://github.com/rivex-dev/rivex-go.git
cd rivex-go

# Run tests
go test ./tests/ -v

# Run linter
golangci-lint run

# Run vet
go vet ./...

# Format code
gofmt -w .
```

## Development Workflow

1. **Create a branch** from `main`
2. **Write tests** for your changes in `tests/rivex_test.go`
3. **Run the full check suite**:
   ```bash
   gofmt -l .
   go vet ./...
   golangci-lint run
   go test ./tests/ -v -race
   ```
4. **Open a pull request** targeting `main`

## Writing Tests

- All tests go in `tests/rivex_test.go` (package `rivex_test`)
- Use `testing.T` and standard Go test conventions
- Use `TestClient` for handler integration tests -- no real server needed
- Use table-driven tests where appropriate
- Run with `-race` flag to detect data races

```go
func TestMyFeature(t *testing.T) {
    app := rivex.New()

    app.Get("/test", func(ctx *rivex.Context) error {
        return ctx.SendJSON(map[string]string{"result": "ok"})
    })

    client := rivex.NewTestClient(app)
    resp := client.Get("/test")

    if resp.StatusCode != 200 {
        t.Fatalf("expected 200, got %d", resp.StatusCode)
    }
}
```

## Commit Messages

Use conventional commit style:

```
fix(middleware): strip Content-Length after gzip compression
feat(router): add wildcard path matching
test(security): add tests for OAuth2 token extraction
docs(readme): update quick start example
```

## Versioning

This project uses [Semantic Versioning](https://semver.org/).
Version is tracked in `go.mod` and `CHANGELOG.md`.

## Code Style

- Format with `gofmt` (or `goimports`)
- Lint with `golangci-lint`
- Vet with `go vet`
- Exported types and functions must have doc comments
- Use functional options pattern (`WithXxx`) for configuration
- Return errors rather than panicking
- Use `interface{}` (not `any`) for Go 1.22 compatibility across all toolchains

## CI

GitHub Actions runs on every push and PR:

| Check | Command |
|-------|---------|
| Format | `gofmt -l .` |
| Vet | `go vet ./...` |
| Lint | `golangci-lint run` |
| Test | `go test ./tests/ -v -race` |
| Build | `go build ./...` |

All checks must pass before merging.
