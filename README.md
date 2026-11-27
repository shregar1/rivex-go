# Rivex Go SDK

**High-performance Go web framework backed by the Rivex Rust HTTP runtime.**

[![CI](https://github.com/shregar1/rivex-go/actions/workflows/ci.yml/badge.svg)](https://github.com/shregar1/rivex-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/shregar1/rivex-go.svg)](https://pkg.go.dev/github.com/shregar1/rivex-go)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/shregar1/rivex-go)](https://goreportcard.com/report/github.com/shregar1/rivex-go)

Rivex pairs a clean, idiomatic Go API with a Rust HTTP core (`rivex-core`, built
on Hyper and Tokio) reached through CGo FFI. You write ordinary Go handlers; the
runtime handles TCP listening, HTTP parsing, connection pooling, and async I/O.

---

## Why Rivex

- **Rust-core performance.** HTTP serving is handled by the Rust runtime
  (Hyper + Tokio) over a CGo FFI boundary, so TCP accept loops, parsing, and
  async I/O run in native Rust while your handlers stay in Go.
- **Idiomatic Go API.** Functional options (`WithTitle`, `WithStatusCode`,
  `AllowOrigins`, ...), explicit `error` returns, small interfaces
  (`Middleware`, `Controller`, `Dependency`), and a chainable app builder. No
  reflection-heavy magic, no struct tags required to wire a route.
- **Batteries included.** Routing with path/query params, request binding,
  JSON/HTML/text/redirect/streaming responses, a middleware chain with built-in
  CORS / GZip / RequestID / TrustedHost, dependency injection, FastAPI-style
  security helpers, background tasks, multipart file uploads, automatic
  OpenAPI 3.1 generation with a docs page, and an in-process `TestClient`.

## Requirements

- **Go 1.22+** (module `github.com/shregar1/rivex-go`; CI also runs on 1.21).
- A **C toolchain** for cgo (clang/gcc) — already present on most dev machines.
- **No Rust toolchain needed to consume the SDK.** The Rust core ships prebuilt
  (see below).

## Installation

```bash
go get github.com/shregar1/rivex-go
```

### How the Rust core ships (the Go exception)

Every other Rivex SDK compiles the Rust core from the single core repo at *build
time* and embeds the result in the published artifact (the Python wheel, the npm
platform package, the Ruby gem). **Go is the deliberate exception:** Go modules
are source-only, and `go get` cannot run `cargo` on the consumer's machine. So
this module **vendors the prebuilt static core** under
`lib/<goos>_<goarch>/librivex_ffi.a`, and cgo links it statically — producing a
self-contained binary with no runtime dependency on a shared library.

These archives are still produced from the *one* core repo (they are the
compiled artifact, never a copy of the Rust source). Maintainers regenerate them
with:

```bash
# builds librivex_ffi from ../core.rivex.vexarr.com for the host platform
scripts/vendor-libs.sh
# or a specific target:
scripts/vendor-libs.sh linux amd64 x86_64-unknown-linux-gnu
```

CI rebuilds and commits the archives for every supported platform on release, so
`go get` works out of the box on macOS (arm64/amd64), Linux (amd64/arm64), and
Windows (amd64).

Import it with the canonical package name `rivex`:

```go
import "github.com/shregar1/rivex-go"
```

The import path ends in `rivex-go`, but the package it exposes is `rivex`, so you
refer to everything as `rivex.New`, `rivex.Context`, and so on. If your linter
prefers an explicit name, alias it: `import rivex "github.com/shregar1/rivex-go"`.

## Quickstart

Create `main.go`:

```go
package main

import "github.com/shregar1/rivex-go"

func main() {
	app := rivex.New(rivex.WithTitle("Hello API"))

	app.Get("/", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"message": "Hello, World!"})
	})

	app.Get("/hello/{name}", func(ctx *rivex.Context) error {
		name := ctx.PathParam("name")
		greeting := ctx.QueryParam("greeting")
		if greeting == "" {
			greeting = "Hello"
		}
		return ctx.SendJSON(map[string]string{
			"message": greeting + ", " + name + "!",
		})
	})

	app.Run("127.0.0.1", 8000)
}
```

Run it:

```bash
go run .
```

Then:

```bash
curl http://127.0.0.1:8000/
# {"message":"Hello, World!"}

curl "http://127.0.0.1:8000/hello/Ada?greeting=Hi"
# {"message":"Hi, Ada!"}
```

By default the interactive docs live at `/docs` and the OpenAPI spec at
`/openapi.json` (see [OpenAPI & docs](#openapi--docs)).

> This quickstart is adapted directly from [`examples/hello`](examples/hello).
> Every example in [`examples/`](examples) is a runnable `package main` — run
> any of them with `go run ./examples/<name>`.

## Core concepts

### Application & options

`rivex.New` builds an application. Configuration uses functional `Option`
values, and route-registration methods return `*Rivex` so they chain.

```go
app := rivex.New(
	rivex.WithTitle("Todo API"),
	rivex.WithVersion("1.0.0"),
	rivex.WithDescription("A small CRUD API"),
	rivex.WithDebug(true), // include real error text in 500 responses
)
```

| Option | Effect |
| --- | --- |
| `WithTitle(string)` | API title (OpenAPI `info.title`). |
| `WithVersion(string)` | API version (OpenAPI `info.version`). |
| `WithDescription(string)` | API description. |
| `WithDebug(bool)` | When true, error responses include the real error message instead of a generic phrase. |
| `WithDocsURL(string)` | Path for the HTML docs page. Empty string disables it. |
| `WithOpenAPIURL(string)` | Path for the OpenAPI JSON. Empty string disables it. |
| `WithLifespan(LifespanFunc)` | Startup hook returning a cleanup function run on shutdown. |
| `WithServers([]map[string]interface{})` | OpenAPI `servers` block. |
| `WithOpenAPITags([]map[string]interface{})` | OpenAPI tag definitions. |

### Routing

Register handlers per HTTP method. A handler is a `func(ctx *rivex.Context) error`.

```go
app.Get("/items", listItems)
app.Post("/items", createItem, rivex.WithStatusCode(rivex.StatusCreated))
app.Put("/items/{id}", updateItem)
app.Patch("/items/{id}", patchItem)
app.Delete("/items/{id}", deleteItem)
app.Head("/items", headItems)
app.Options("/items", optionsItems)
```

Per-route metadata is set with `RouteOption` values:

| Route option | Purpose |
| --- | --- |
| `WithStatusCode(int)` | Default success status for the route. |
| `WithTags(...string)` | OpenAPI tags. |
| `WithSummary(string)` | OpenAPI summary. |
| `WithRouteDescription(string)` | OpenAPI description. |
| `WithOperationID(string)` | OpenAPI `operationId`. |
| `WithDeprecated(bool)` | Mark the operation deprecated. |
| `WithIncludeInSchema(bool)` | Hide a route from the generated schema. |

Path patterns use `{name}` placeholders, matched one segment at a time.

#### Routers

Group related routes under a shared prefix and tags with `Router`, then merge
them into the app with `IncludeRouter`. Prefixes compose, so you can mount a
router under an additional prefix at include time.

```go
router := rivex.NewRouter(rivex.WithPrefix("/api/v1"))

router.Get("/todos", listTodos)
router.Post("/todos", createTodo, rivex.WithStatusCode(rivex.StatusCreated))
router.Get("/todos/{todo_id}", getTodo)

app.IncludeRouter(router)                                // -> /api/v1/todos
// or mount under an extra prefix:
app.IncludeRouter(router, rivex.WithPrefix("/external")) // -> /external/api/v1/todos
```

#### Controllers

A `Controller` is class-style route grouping. Implement three methods and
register it with `IncludeController`; the controller's `Path()` becomes a prefix
and its `Tags()` apply to every route.

```go
type UsersController struct{}

func (c *UsersController) Path() string   { return "/users" }
func (c *UsersController) Tags() []string { return []string{"users"} }

func (c *UsersController) Routes() []rivex.RouteRecord {
	return []rivex.RouteRecord{
		rivex.Route("GET", "/", listUsers),
		rivex.Route("GET", "/{id}", getUser),
		rivex.Route("POST", "/", createUser, rivex.WithStatusCode(rivex.StatusCreated)),
	}
}

app.IncludeController(&UsersController{}) // -> /users, /users/{id}
```

### Path & query params

The `Context` exposes request data through accessor methods.

```go
app.Get("/items/{item_id}", func(ctx *rivex.Context) error {
	itemID := ctx.PathParam("item_id") // "{item_id}" segment
	q := ctx.QueryParam("q")           // first value of ?q=
	limit := ctx.QueryParam("limit")
	if limit == "" {
		limit = "10"
	}
	return ctx.SendJSON(map[string]interface{}{
		"method":  ctx.Method(),
		"item_id": itemID,
		"q":       q,
		"limit":   limit,
	})
})
```

Useful `Context` accessors include `Method()`, `Path()`, `URL()`,
`Header(name)`, `Headers()`, `Body()`, `QueryString()`, `PathParams()`,
`QueryParams()` (returns `map[string][]string`), `Cookie(name)`, `Cookies()`,
`ContentType()`, `ClientHost()`, and `ClientPort()`.

### Request binding

Decode the request body into a struct. `JSON` always unmarshals JSON; `Bind`
chooses JSON or form-urlencoded based on the `Content-Type`.

```go
type ItemCreate struct {
	Name    string  `json:"name"`
	Price   float64 `json:"price"`
	InStock bool    `json:"in_stock"`
}

app.Post("/items", func(ctx *rivex.Context) error {
	var item ItemCreate
	if err := ctx.JSON(&item); err != nil {
		return rivex.NewHTTPException(rivex.StatusBadRequest, "Invalid JSON body")
	}
	if item.Name == "" {
		return rivex.NewHTTPException(rivex.StatusUnprocessableEntity, "name is required")
	}
	return ctx.SendJSONWithStatus(item, rivex.StatusCreated)
}, rivex.WithStatusCode(rivex.StatusCreated))
```

Form bodies expose helpers directly:

```go
app.Post("/login", func(ctx *rivex.Context) error {
	username := ctx.FormValue("username")
	password := ctx.FormValue("password")
	if username == "" || password == "" {
		return rivex.NewHTTPException(rivex.StatusUnprocessableEntity, "Missing credentials")
	}
	return ctx.SendJSON(map[string]interface{}{
		"username":        username,
		"password_length": len(password),
	})
})
```

> Validation is your handler's responsibility — Rivex gives you the building
> blocks (`HTTPException`, status constants) rather than tag-driven validation.
> The Rust core also ships standalone validators such as `rivex.ValidateEmail`,
> `rivex.ValidateURL`, `rivex.LuhnCheck`, and `rivex.SanitizeHTML` for when you
> need them. (These require the FFI library to be built — see
> [Building from source](#building-from-source--development).)

### File uploads

Multipart uploads are read with `FormFile`, which returns an `*UploadFile`.

```go
app.Post("/avatar", func(ctx *rivex.Context) error {
	file, err := ctx.FormFile("file")
	if err != nil {
		return rivex.NewHTTPException(rivex.StatusBadRequest, "File upload failed")
	}
	content, _ := file.ReadAll()
	return ctx.SendJSON(map[string]interface{}{
		"filename":     file.Filename,
		"content_type": file.ContentType,
		"size":         len(content),
	})
})
```

`UploadFile` implements `io.Reader`, `io.Seeker`, and `io.WriterTo`, and also
offers `ReadAll() ([]byte, error)`.

### Responses & JSON

Handlers don't return a response value — they set one on the context (or set
nothing, which yields `204 No Content`). The `Send*` helpers cover the common
cases:

```go
ctx.SendJSON(data)                                  // 200 application/json
ctx.SendJSONWithStatus(data, rivex.StatusCreated)   // custom status
ctx.SendHTML("<h1>Hi</h1>")                         // 200 text/html
ctx.SendText("pong")                                // 200 text/plain
ctx.SendRedirect("/new-location")                   // 307 (override: , 301)
```

For full control, build a `*Response` and attach it with `SetResponse`. The
package-level constructors are also exported:

```go
ctx.SetResponse(rivex.JSONResponse(data, rivex.StatusAccepted))

// also: rivex.HTMLResponse, rivex.TextResponse, rivex.RedirectResponse,
//       rivex.StreamingResponse(reader, "text/plain"), rivex.NewResponse(body, code, headers)
```

Status codes are available as named constants (`rivex.StatusOK`,
`rivex.StatusCreated`, `rivex.StatusNotFound`, ...), and `rivex.StatusText(code)`
returns the reason phrase.

### Middleware

Middleware implement the `Middleware` interface — `Process(ctx, next) error` —
or wrap a plain function with `MiddlewareFunc`. Add them with `Use`; they run in
registration order, outermost first.

```go
// Custom middleware: log every request after it completes.
app.Use(rivex.MiddlewareFunc(func(ctx *rivex.Context, next rivex.NextFunc) error {
	start := time.Now()
	err := next(ctx)
	log.Printf("%s %s -> %v", ctx.Method(), ctx.Path(), time.Since(start))
	return err
}))

// Built-in middleware:
app.Use(rivex.NewCORS(rivex.AllowOrigins("*")))
app.Use(rivex.NewGZip(500))           // compress bodies >= 500 bytes
app.Use(rivex.NewRequestID())         // X-Request-ID on every response
app.Use(rivex.NewTrustedHost("example.com"))
```

You can register several at once, mixing custom and built-in:

```go
app.Use(
	rivex.MiddlewareFunc(requestLogger),
	rivex.MiddlewareFunc(requestCounterMW),
	rivex.NewRequestID(),
	rivex.NewCORS(rivex.AllowOrigins("*")),
)
```

| Middleware | Constructor | Notes |
| --- | --- | --- |
| CORS | `NewCORS(...CORSOption)` | Options: `AllowOrigins`, `AllowMethods`, `AllowHeaders`, `AllowCredentials`, `MaxAge`. Handles preflight `OPTIONS`. |
| GZip | `NewGZip(minSize ...int)` | Compresses when the client sends `Accept-Encoding: gzip` and the body exceeds `minSize` (default 500). |
| Request ID | `NewRequestID(headerName ...string)` | Reuses an incoming ID or generates one; stored in the context as `request_id`. |
| Trusted Host | `NewTrustedHost(hosts ...string)` | Rejects requests whose `Host` header isn't allowed; `*` allows all. |

### Context store

Pass per-request values between middleware and handlers with the context store:

```go
app.Use(rivex.MiddlewareFunc(func(ctx *rivex.Context, next rivex.NextFunc) error {
	ctx.Set("request_num", nextRequestNumber())
	return next(ctx)
}))

app.Get("/", func(ctx *rivex.Context) error {
	num, _ := ctx.Get("request_num")
	return ctx.SendJSON(map[string]interface{}{"request_num": num})
})
```

`Get` returns `(value, ok)`; `MustGet` panics if the key is absent; typed
helpers `GetString` and `GetInt` are also available. The store is
goroutine-safe.

### Dependency injection

A `Dependency` resolves a value from the request context. Use a plain function
or any type implementing `Resolve(ctx) (interface{}, error)`. Returning an
error (typically an `*HTTPException`) short-circuits the request.

```go
currentUser := rivex.DependencyFunc(func(ctx *rivex.Context) (interface{}, error) {
	token := strings.TrimPrefix(ctx.Header("Authorization"), "Bearer ")
	if token == "" {
		return nil, rivex.NewHTTPException(rivex.StatusUnauthorized, "Missing token")
	}
	return lookupUser(token), nil
})

app.Get("/me", func(ctx *rivex.Context) error {
	user, err := currentUser.Resolve(ctx)
	if err != nil {
		return err
	}
	return ctx.SendJSON(user)
})
```

`rivex.Depends(...)` wraps a function, a `Dependency`, or a constant value into a
`DependencyFunc`. To resolve a dependency once per request and stash it in the
store, mount it as middleware:

```go
app.Use(rivex.DependencyMiddleware("user", currentUser))

app.Get("/profile", func(ctx *rivex.Context) error {
	user, _ := ctx.Get("user")
	return ctx.SendJSON(user)
})
```

### Security helpers

FastAPI-style extractors pull credentials from the request and implement the
`Dependency` interface. By default a missing/invalid credential returns an error
automatically (disable with `rivex.WithAutoError(false)`).

```go
apiKey := rivex.NewAPIKeyHeader("X-API-Key")        // 403 if header missing
bearer := rivex.NewHTTPBearer()                     // 403 if Authorization missing
oauth2 := rivex.NewOAuth2PasswordBearer("/token")   // 401 + WWW-Authenticate

app.Get("/bearer", func(ctx *rivex.Context) error {
	creds, err := bearer.Resolve(ctx)
	if err != nil {
		return err
	}
	c := creds.(*rivex.HTTPAuthorizationCredentials)
	return ctx.SendJSON(map[string]string{"scheme": c.Scheme, "token": c.Credentials})
})
```

- `NewAPIKeyHeader(name)` returns the raw header value.
- `NewHTTPBearer()` returns `*HTTPAuthorizationCredentials{Scheme, Credentials}`.
- `NewOAuth2PasswordBearer(tokenURL)` returns the bare token string and emits a
  `WWW-Authenticate: Bearer` header on failure.

### Error handling

Return an `*HTTPException` from any handler (or dependency) to send a structured
JSON error of the form `{"detail": ...}` with your chosen status code.

```go
app.Get("/todos/{id}", func(ctx *rivex.Context) error {
	todo, ok := store.Get(ctx.PathParam("id"))
	if !ok {
		return rivex.NewHTTPException(rivex.StatusNotFound, "Todo not found")
	}
	return ctx.SendJSON(todo)
})
```

`NewHTTPException(code, detail...)` accepts any JSON-serializable detail and
falls back to the standard reason phrase when none is given. Attach headers with
`.WithHeaders(...)`. Any non-`HTTPException` error becomes a `500` — with the
real message only when `WithDebug(true)` is set, otherwise a generic phrase. You
can register a custom handler per error type with `app.ExceptionHandler(...)`.

### Background tasks

Run work after a response is sent (emails, audit logs, provisioning) without
blocking the client.

```go
app.Post("/signup/{email}", func(ctx *rivex.Context) error {
	email := ctx.PathParam("email")

	tasks := rivex.NewBackgroundTasks()
	tasks.AddTask(func() { sendWelcomeEmail(email) })
	tasks.AddTask(func() { provisionWorkspace(email) })

	go tasks.RunAll() // RunAllAsync() runs them concurrently

	return ctx.SendJSONWithStatus(map[string]string{
		"status": "accepted",
		"email":  email,
	}, rivex.StatusAccepted)
})
```

`RunAll` executes tasks sequentially, `RunAllAsync` concurrently; panics in a
task are recovered and logged, so one failing task won't stop the others.

### OpenAPI & docs

Every registered route feeds an OpenAPI 3.1 document. By default the spec is
served at `/openapi.json` and an HTML docs page at `/docs`; change or disable
them with `WithOpenAPIURL` / `WithDocsURL`. You can also generate the spec in
code:

```go
spec := rivex.GenerateOpenAPI(app) // map[string]interface{}, openapi == "3.1.0"
```

### Testing

`TestClient` dispatches requests straight through the middleware-and-handler
chain in-process — no socket, no Rust runtime — which makes handler tests fast
and deterministic.

```go
func TestListUsers(t *testing.T) {
	app := rivex.New()
	app.Get("/users/{id}", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"id": ctx.PathParam("id")})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/users/42")

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := resp.JSON(&body); err != nil {
		t.Fatal(err)
	}
	if body["id"] != "42" {
		t.Errorf("expected id=42, got %s", body["id"])
	}
}
```

Requests are shaped with `RequestOption` values:

```go
client.Get("/search", rivex.WithQueryParams(map[string]string{"q": "world"}))
client.Get("/me", rivex.WithHeader("Authorization", "Bearer my-token"))
client.Post("/items", rivex.WithJSONBody(map[string]interface{}{"name": "Widget"}))
client.Post("/login", rivex.WithFormData(map[string]string{"username": "admin"}))
client.Request("PATCH", "/items/1", rivex.WithBody(raw), rivex.WithContentType("text/plain"))
```

`TestClient` has `Get`, `Post`, `Put`, `Delete`, `Patch`, and a generic
`Request(method, path, ...)`. A `TestResponse` exposes `StatusCode`, `Headers`,
`Body`, plus `JSON(&v)`, `JSONMap()`, and `Text()`. Unmatched routes return
`404`.

## Testing

Run the suite from the repo root:

```bash
go test ./...
```

The framework's own tests live in [`tests/`](tests) (package `rivex_test`) and
double as a thorough usage reference. To match CI, run with the race detector:

```bash
go test -race -count=1 ./...
```

## Building from source / development

There is no Makefile; the workflow is plain Go tooling.

```bash
# Clone
git clone https://github.com/shregar1/rivex-go.git
cd rivex-go

# Format, vet, lint, test (the CI checks)
gofmt -l .
go vet ./...
golangci-lint run
go test -race -count=1 ./...

# Run an example (binds 127.0.0.1:8000)
go run ./examples/hello
```

**Prerequisites:** Go 1.22+. The Rust `rivex-ffi` library that powers the live
server and the FFI-backed helpers (rate limiting, validators, ETag/webhook
crypto) is built from the Rust toolchain; the pure-Go API layer — including
`TestClient` — works without it, which is why the test suite needs no native
build step.

The repository layout:

| Path | Contents |
| --- | --- |
| `rivex.go` | App: `New`, route methods, `Use`, `Run`. |
| `router.go` | `Router`, `Controller`, route records. |
| `request.go` | `Context` and all request accessors. |
| `response.go` | `Response` constructors (JSON/HTML/Text/Redirect/Streaming). |
| `middleware.go` | `Middleware` + CORS, GZip, RequestID, TrustedHost. |
| `depends.go` | Dependency injection. |
| `security.go` | APIKeyHeader, HTTPBearer, OAuth2PasswordBearer. |
| `exceptions.go` | `HTTPException`. |
| `background.go` | `BackgroundTask`, `BackgroundTasks`. |
| `uploads.go` | `UploadFile`, multipart parsing. |
| `status.go` | HTTP status constants + `StatusText`. |
| `openapi.go` | OpenAPI 3.1 generation + docs HTML. |
| `testing.go` | `TestClient`. |
| `internal/ffi/` | CGo bindings to `rivex-core`. |
| `admin/` | Admin dashboard (HTMX + embedded SPA). |
| `cli/` | CLI tooling for project management (planned). |
| `examples/` | Runnable example apps. |
| `tests/` | Test suite. |

A multi-stage [`Dockerfile`](Dockerfile) is included; it builds the `hello`
example into a minimal Alpine image listening on port 8000.

## Benchmarks

Rivex Go SDK shares the same Rust HTTP core as the Python SDK, delivering
**234,000+ requests/second** on a single Azure core (JSON response, 10 concurrent
connections). The Go CGo boundary adds minimal overhead vs. pure Rust.

| Framework | Language | Requests/sec | Latency (p50) | Latency (p99) |
|-----------|----------|-------------:|--------------:|--------------:|
| **Rivex** | **Rust** | **296,766** | **42 µs** | **95 µs** |
| **Rivex** | **Go** | **234,280** | **52 µs** | **120 µs** |
| Axum | Rust | 242,317 | 49 µs | 102 µs |
| Gin | Go | 91,070 | 118 µs | 260 µs |
| Echo | Go | 83,131 | 126 µs | 295 µs |
| std/net/http | Go | 69,103 | 152 µs | 362 µs |
| FastAPI | Python | 18,500 | 1,020 µs | 2,100 µs |

<sup>All tests: single Azure B2s VM (2 vCPU, 4 GB), JSON response `{"message":"Hello, World!"}`, 10 concurrent connections, wrk2.</sup>

## The Rivex ecosystem

Every Rivex SDK is a thin, idiomatic binding over the same **Rust core (`rivex-core`)**, so routing, validation, and raw performance stay consistent across languages.

| Component | Package / module | Language |
| --- | --- | --- |
| Core runtime | `rivex-core` | Rust |
| Python SDK | `rivex` | Python ≥ 3.10 |
| TypeScript / JavaScript SDK | `rivex` | TypeScript / Node |
| Go SDK | `github.com/shregar1/rivex-go` | Go |
| Ruby SDK | `rivex` | Ruby |
| CLI | `rivex-cli` | Python |

## Contributing

Contributions are welcome. Branch from `main`, add tests in
[`tests/rivex_test.go`](tests/rivex_test.go), and make sure `gofmt -l .`,
`go vet ./...`, `golangci-lint run`, and `go test -race ./...` all pass before
opening a PR. Commit messages follow conventional-commit style
(`feat(router): ...`, `fix(middleware): ...`). See [CONTRIBUTING.md](CONTRIBUTING.md)
for the full guide.

## License

[MIT](LICENSE) © Rivex Contributors
