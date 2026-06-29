# Changelog

All notable changes to this project will be documented in this file.

This project follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.0] - 2025-05-10

### Added
- Initial release
- `Rivex` application with functional options (`WithTitle`, `WithVersion`, etc.)
- HTTP method routing: `Get`, `Post`, `Put`, `Delete`, `Patch`, `Head`, `Options`
- `Router` for route grouping with prefix and tag support
- `Controller` interface for class-style route grouping
- `Context` type with request accessors (path params, query params, headers, cookies, body parsing)
- Response constructors: `JSONResponse`, `HTMLResponse`, `TextResponse`, `RedirectResponse`, `StreamingResponse`
- Middleware system with `Middleware` interface and `MiddlewareFunc` adapter
- Built-in middleware: `CORSMiddleware`, `GZipMiddleware`, `RequestIDMiddleware`, `TrustedHostMiddleware`
- Dependency injection with `Dependency` interface, `DependencyFunc`, and `Depends()`
- Security helpers: `APIKeyHeader`, `HTTPBearer`, `OAuth2PasswordBearer`
- `HTTPException` with default status phrases and custom headers
- `BackgroundTask` and `BackgroundTasks` for post-response async work
- `UploadFile` type for multipart file upload handling
- HTTP status code constants matching RFC 9110
- `TestClient` for in-process request testing without a server
- OpenAPI 3.1 spec generation from registered routes
- CGo FFI binding placeholders for future `rivex-ffi` Rust crate
- Examples: hello world, todo API, request features demo

[Unreleased]: https://github.com/rivex-dev/rivex-go/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/rivex-dev/rivex-go/releases/tag/v0.1.0
