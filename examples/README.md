# Rivex Go SDK — Examples

Each subdirectory is a runnable `package main`. From the repo root:

```sh
go run ./examples/hello
```

| Example | What it shows |
| --- | --- |
| `hello/` | Minimal app: routing, path params, query params, JSON response. |
| `todo_api/` | CRUD with `Router`, `CORSMiddleware`, `HTTPException`, status codes. |
| `request_features/` | Dependency injection, custom middleware, file/form parsing, controllers. |
| `auth_bearer/` | `HTTPBearer` security helper guarding protected routes. |
| `background_tasks/` | Kicking off work after the response is sent with `BackgroundTasks`. |
| `middleware_chain/` | Stacking custom + built-in middleware (logger, counter, request id, CORS). |
| `testing_demo/` | Using `TestClient` to exercise an app without starting a server. |

Examples bind to `127.0.0.1:8000` by default. Stop with Ctrl+C.
