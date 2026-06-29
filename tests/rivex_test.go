package rivex_test

import (
	"encoding/json"
	"strings"
	"testing"

	rivex "github.com/rivex-dev/rivex-go"
)

// --- Application Tests ---

func TestNewApp(t *testing.T) {
	app := rivex.New()
	if app == nil {
		t.Fatal("expected non-nil app")
	}
}

func TestNewAppWithOptions(t *testing.T) {
	app := rivex.New(
		rivex.WithTitle("Test API"),
		rivex.WithVersion("1.0.0"),
		rivex.WithDescription("A test API"),
		rivex.WithDebug(true),
	)
	if app == nil {
		t.Fatal("expected non-nil app")
	}
}

func TestRouteRegistration(t *testing.T) {
	app := rivex.New()
	handler := func(ctx *rivex.Context) error { return nil }

	app.Get("/get", handler)
	app.Post("/post", handler)
	app.Put("/put", handler)
	app.Delete("/delete", handler)
	app.Patch("/patch", handler)
	app.Head("/head", handler)
	app.Options("/options", handler)

	routes := app.GetRoutes()
	if len(routes) != 7 {
		t.Fatalf("expected 7 routes, got %d", len(routes))
	}

	expectedMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	for i, route := range routes {
		if route.Method != expectedMethods[i] {
			t.Errorf("route %d: expected method %s, got %s", i, expectedMethods[i], route.Method)
		}
	}
}

func TestRouteWithOptions(t *testing.T) {
	app := rivex.New()
	handler := func(ctx *rivex.Context) error { return nil }

	app.Post("/items", handler,
		rivex.WithStatusCode(201),
		rivex.WithTags("items"),
		rivex.WithSummary("Create item"),
		rivex.WithRouteDescription("Creates a new item"),
		rivex.WithOperationID("createItem"),
	)

	routes := app.GetRoutes()
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}

	route := routes[0]
	if route.StatusCode != 201 {
		t.Errorf("expected status 201, got %d", route.StatusCode)
	}
	if len(route.Tags) != 1 || route.Tags[0] != "items" {
		t.Errorf("expected tags [items], got %v", route.Tags)
	}
	if route.Summary != "Create item" {
		t.Errorf("expected summary 'Create item', got %q", route.Summary)
	}
}

func TestChainableAPI(t *testing.T) {
	handler := func(ctx *rivex.Context) error { return nil }
	app := rivex.New().
		Get("/a", handler).
		Post("/b", handler).
		Put("/c", handler)

	routes := app.GetRoutes()
	if len(routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(routes))
	}
}

// --- Router Tests ---

func TestRouter(t *testing.T) {
	router := rivex.NewRouter(rivex.WithPrefix("/api/v1"))
	handler := func(ctx *rivex.Context) error { return nil }

	router.Get("/users", handler)
	router.Post("/users", handler)

	routes := router.Routes()
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}

	if routes[0].Path != "/api/v1/users" {
		t.Errorf("expected /api/v1/users, got %s", routes[0].Path)
	}
}

func TestIncludeRouter(t *testing.T) {
	app := rivex.New()
	router := rivex.NewRouter(rivex.WithPrefix("/api"))
	handler := func(ctx *rivex.Context) error { return nil }

	router.Get("/items", handler, rivex.WithTags("items"))

	app.IncludeRouter(router)

	routes := app.GetRoutes()
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if routes[0].Path != "/api/items" {
		t.Errorf("expected /api/items, got %s", routes[0].Path)
	}
}

func TestIncludeRouterWithPrefix(t *testing.T) {
	app := rivex.New()
	router := rivex.NewRouter(rivex.WithPrefix("/items"))
	handler := func(ctx *rivex.Context) error { return nil }

	router.Get("/", handler)
	router.Get("/{id}", handler)

	app.IncludeRouter(router, rivex.WithPrefix("/api/v1"))

	routes := app.GetRoutes()
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}
	if routes[0].Path != "/api/v1/items" {
		t.Errorf("expected /api/v1/items, got %s", routes[0].Path)
	}
}

// --- Controller Tests ---

type usersController struct{}

func (c *usersController) Path() string    { return "/users" }
func (c *usersController) Tags() []string  { return []string{"users"} }
func (c *usersController) Routes() []rivex.RouteRecord {
	return []rivex.RouteRecord{
		rivex.Route("GET", "/", func(ctx *rivex.Context) error {
			return ctx.SendJSON([]string{"user1", "user2"})
		}),
		rivex.Route("GET", "/{id}", func(ctx *rivex.Context) error {
			return ctx.SendJSON(map[string]string{"id": ctx.PathParam("id")})
		}),
		rivex.Route("POST", "/", func(ctx *rivex.Context) error {
			return ctx.SendJSONWithStatus(map[string]string{"created": "true"}, 201)
		}, rivex.WithStatusCode(201)),
	}
}

func TestIncludeController(t *testing.T) {
	app := rivex.New()
	app.IncludeController(&usersController{})

	routes := app.GetRoutes()
	if len(routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(routes))
	}
	if routes[0].Path != "/users" {
		t.Errorf("expected /users, got %s", routes[0].Path)
	}
	if routes[1].Path != "/users/{id}" {
		t.Errorf("expected /users/{id}, got %s", routes[1].Path)
	}
}

// --- Context Tests ---

func TestContextPathParams(t *testing.T) {
	app := rivex.New()
	app.Get("/users/{id}", func(ctx *rivex.Context) error {
		id := ctx.PathParam("id")
		return ctx.SendJSON(map[string]string{"id": id})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/users/42")

	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := resp.JSON(&body); err != nil {
		t.Fatal(err)
	}
	if body["id"] != "42" {
		t.Errorf("expected id=42, got %s", body["id"])
	}
}

func TestContextQueryParams(t *testing.T) {
	app := rivex.New()
	app.Get("/search", func(ctx *rivex.Context) error {
		q := ctx.QueryParam("q")
		page := ctx.QueryParam("page")
		return ctx.SendJSON(map[string]string{"q": q, "page": page})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/search?q=hello&page=2")

	var body map[string]string
	if err := resp.JSON(&body); err != nil {
		t.Fatal(err)
	}
	if body["q"] != "hello" {
		t.Errorf("expected q=hello, got %s", body["q"])
	}
	if body["page"] != "2" {
		t.Errorf("expected page=2, got %s", body["page"])
	}
}

func TestContextQueryParamsOption(t *testing.T) {
	app := rivex.New()
	app.Get("/search", func(ctx *rivex.Context) error {
		q := ctx.QueryParam("q")
		return ctx.SendJSON(map[string]string{"q": q})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/search", rivex.WithQueryParams(map[string]string{"q": "world"}))

	var body map[string]string
	if err := resp.JSON(&body); err != nil {
		t.Fatal(err)
	}
	if body["q"] != "world" {
		t.Errorf("expected q=world, got %s", body["q"])
	}
}

func TestContextHeaders(t *testing.T) {
	app := rivex.New()
	app.Get("/echo-header", func(ctx *rivex.Context) error {
		auth := ctx.Header("Authorization")
		return ctx.SendJSON(map[string]string{"auth": auth})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/echo-header", rivex.WithHeader("Authorization", "Bearer test-token"))

	var body map[string]string
	if err := resp.JSON(&body); err != nil {
		t.Fatal(err)
	}
	if body["auth"] != "Bearer test-token" {
		t.Errorf("expected auth=Bearer test-token, got %s", body["auth"])
	}
}

func TestContextJSONBody(t *testing.T) {
	app := rivex.New()
	app.Post("/items", func(ctx *rivex.Context) error {
		var item struct {
			Name  string  `json:"name"`
			Price float64 `json:"price"`
		}
		if err := ctx.JSON(&item); err != nil {
			return rivex.NewHTTPException(400, "Invalid JSON")
		}
		return ctx.SendJSON(item)
	})

	client := rivex.NewTestClient(app)
	resp := client.Post("/items", rivex.WithJSONBody(map[string]interface{}{
		"name":  "Widget",
		"price": 9.99,
	}))

	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d: %s", resp.StatusCode, resp.Text())
	}

	var body map[string]interface{}
	if err := resp.JSON(&body); err != nil {
		t.Fatal(err)
	}
	if body["name"] != "Widget" {
		t.Errorf("expected name=Widget, got %v", body["name"])
	}
}

func TestContextStore(t *testing.T) {
	app := rivex.New()
	app.Get("/store", func(ctx *rivex.Context) error {
		ctx.Set("user", "alice")
		user, ok := ctx.Get("user")
		if !ok {
			return rivex.NewHTTPException(500, "user not found in store")
		}
		return ctx.SendJSON(map[string]string{"user": user.(string)})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/store")

	var body map[string]string
	if err := resp.JSON(&body); err != nil {
		t.Fatal(err)
	}
	if body["user"] != "alice" {
		t.Errorf("expected user=alice, got %s", body["user"])
	}
}

func TestContextFormData(t *testing.T) {
	app := rivex.New()
	app.Post("/login", func(ctx *rivex.Context) error {
		username := ctx.FormValue("username")
		password := ctx.FormValue("password")
		return ctx.SendJSON(map[string]interface{}{
			"username":        username,
			"password_length": len(password),
		})
	})

	client := rivex.NewTestClient(app)
	resp := client.Post("/login", rivex.WithFormData(map[string]string{
		"username": "admin",
		"password": "secret123",
	}))

	var body map[string]interface{}
	if err := resp.JSON(&body); err != nil {
		t.Fatal(err)
	}
	if body["username"] != "admin" {
		t.Errorf("expected username=admin, got %v", body["username"])
	}
}

// --- Response Tests ---

func TestJSONResponse(t *testing.T) {
	resp := rivex.JSONResponse(map[string]string{"key": "value"})
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	if resp.Headers["Content-Type"] != "application/json" {
		t.Errorf("expected application/json, got %s", resp.Headers["Content-Type"])
	}

	var body map[string]string
	if err := json.Unmarshal(resp.Content, &body); err != nil {
		t.Fatal(err)
	}
	if body["key"] != "value" {
		t.Errorf("expected key=value, got %s", body["key"])
	}
}

func TestJSONResponseWithStatus(t *testing.T) {
	resp := rivex.JSONResponse(map[string]string{"created": "true"}, 201)
	if resp.StatusCode != 201 {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}
}

func TestHTMLResponse(t *testing.T) {
	resp := rivex.HTMLResponse("<h1>Hello</h1>")
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	if resp.Headers["Content-Type"] != "text/html; charset=utf-8" {
		t.Errorf("unexpected content type: %s", resp.Headers["Content-Type"])
	}
	if string(resp.Content) != "<h1>Hello</h1>" {
		t.Errorf("unexpected content: %s", string(resp.Content))
	}
}

func TestTextResponse(t *testing.T) {
	resp := rivex.TextResponse("hello world")
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	if string(resp.Content) != "hello world" {
		t.Errorf("unexpected content: %s", string(resp.Content))
	}
}

func TestRedirectResponse(t *testing.T) {
	resp := rivex.RedirectResponse("/new-location")
	if resp.StatusCode != 307 {
		t.Errorf("expected status 307, got %d", resp.StatusCode)
	}
	if resp.Headers["Location"] != "/new-location" {
		t.Errorf("expected Location=/new-location, got %s", resp.Headers["Location"])
	}
}

func TestRedirectResponseWithStatus(t *testing.T) {
	resp := rivex.RedirectResponse("/moved", 301)
	if resp.StatusCode != 301 {
		t.Errorf("expected status 301, got %d", resp.StatusCode)
	}
}

func TestStreamingResponse(t *testing.T) {
	reader := strings.NewReader("streaming content")
	resp := rivex.StreamingResponse(reader, "text/plain")
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	if string(resp.Content) != "streaming content" {
		t.Errorf("unexpected content: %s", string(resp.Content))
	}
}

// --- Middleware Tests ---

func TestCORSMiddleware(t *testing.T) {
	app := rivex.New()
	app.Use(rivex.NewCORS(
		rivex.AllowOrigins("http://example.com"),
		rivex.AllowMethods("GET", "POST"),
	))

	app.Get("/test", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"ok": "true"})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/test", rivex.WithHeader("Origin", "http://example.com"))

	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if resp.Headers["Access-Control-Allow-Origin"] != "http://example.com" {
		t.Errorf("expected CORS origin header, got %s", resp.Headers["Access-Control-Allow-Origin"])
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	app := rivex.New()
	app.Use(rivex.NewRequestID())

	app.Get("/test", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"ok": "true"})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/test")

	if resp.Headers["X-Request-ID"] == "" {
		t.Error("expected X-Request-ID header to be set")
	}
}

func TestRequestIDMiddlewarePreserves(t *testing.T) {
	app := rivex.New()
	app.Use(rivex.NewRequestID())

	app.Get("/test", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"ok": "true"})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/test", rivex.WithHeader("X-Request-ID", "my-custom-id"))

	if resp.Headers["X-Request-ID"] != "my-custom-id" {
		t.Errorf("expected X-Request-ID=my-custom-id, got %s", resp.Headers["X-Request-ID"])
	}
}

func TestMiddlewareFunc(t *testing.T) {
	app := rivex.New()
	app.Use(rivex.MiddlewareFunc(func(ctx *rivex.Context, next rivex.NextFunc) error {
		ctx.Set("middleware_ran", true)
		return next(ctx)
	}))

	app.Get("/test", func(ctx *rivex.Context) error {
		ran, _ := ctx.Get("middleware_ran")
		return ctx.SendJSON(map[string]interface{}{"middleware_ran": ran})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/test")

	var body map[string]interface{}
	if err := resp.JSON(&body); err != nil {
		t.Fatal(err)
	}
	if body["middleware_ran"] != true {
		t.Error("expected middleware_ran=true")
	}
}

// --- Exception Tests ---

func TestHTTPException(t *testing.T) {
	exc := rivex.NewHTTPException(404, "Not found")
	if exc.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", exc.StatusCode)
	}
	if exc.Detail != "Not found" {
		t.Errorf("expected detail 'Not found', got %v", exc.Detail)
	}
	if !strings.Contains(exc.Error(), "404") {
		t.Errorf("expected error string to contain 404, got %s", exc.Error())
	}
}

func TestHTTPExceptionDefaultDetail(t *testing.T) {
	exc := rivex.NewHTTPException(500)
	if exc.Detail != "Internal Server Error" {
		t.Errorf("expected default detail, got %v", exc.Detail)
	}
}

func TestHTTPExceptionWithHeaders(t *testing.T) {
	exc := rivex.NewHTTPException(401, "Unauthorized").
		WithHeaders(map[string]string{"WWW-Authenticate": "Bearer"})

	if exc.Headers["WWW-Authenticate"] != "Bearer" {
		t.Error("expected WWW-Authenticate header")
	}
}

func TestHTTPExceptionInHandler(t *testing.T) {
	app := rivex.New()
	app.Get("/fail", func(ctx *rivex.Context) error {
		return rivex.NewHTTPException(403, "Forbidden resource")
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/fail")

	if resp.StatusCode != 403 {
		t.Fatalf("expected status 403, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := resp.JSON(&body); err != nil {
		t.Fatal(err)
	}
	if body["detail"] != "Forbidden resource" {
		t.Errorf("expected detail 'Forbidden resource', got %v", body["detail"])
	}
}

// --- TestClient Tests ---

func TestTestClientGet(t *testing.T) {
	app := rivex.New()
	app.Get("/hello", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"message": "hello"})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/hello")

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := resp.JSON(&body); err != nil {
		t.Fatal(err)
	}
	if body["message"] != "hello" {
		t.Errorf("expected message=hello, got %s", body["message"])
	}
}

func TestTestClientPost(t *testing.T) {
	app := rivex.New()
	app.Post("/echo", func(ctx *rivex.Context) error {
		var data map[string]interface{}
		if err := ctx.JSON(&data); err != nil {
			return err
		}
		return ctx.SendJSON(data)
	})

	client := rivex.NewTestClient(app)
	resp := client.Post("/echo", rivex.WithJSONBody(map[string]string{"key": "value"}))

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestTestClientNotFound(t *testing.T) {
	app := rivex.New()
	client := rivex.NewTestClient(app)
	resp := client.Get("/nonexistent")

	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestTestClientWithMultipleRoutes(t *testing.T) {
	app := rivex.New()
	app.Get("/a", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"route": "a"})
	})
	app.Get("/b", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"route": "b"})
	})
	app.Post("/c", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"route": "c"})
	})

	client := rivex.NewTestClient(app)

	resp := client.Get("/a")
	var body map[string]string
	resp.JSON(&body)
	if body["route"] != "a" {
		t.Errorf("expected route=a, got %s", body["route"])
	}

	resp = client.Get("/b")
	resp.JSON(&body)
	if body["route"] != "b" {
		t.Errorf("expected route=b, got %s", body["route"])
	}

	resp = client.Post("/c")
	resp.JSON(&body)
	if body["route"] != "c" {
		t.Errorf("expected route=c, got %s", body["route"])
	}
}

// --- Security Tests ---

func TestAPIKeyHeader(t *testing.T) {
	apiKey := rivex.NewAPIKeyHeader("X-API-Key")
	app := rivex.New()

	app.Get("/protected", func(ctx *rivex.Context) error {
		key, err := apiKey.Resolve(ctx)
		if err != nil {
			return err
		}
		return ctx.SendJSON(map[string]interface{}{"key": key})
	})

	client := rivex.NewTestClient(app)

	// Without API key - should fail
	resp := client.Get("/protected")
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403 without API key, got %d", resp.StatusCode)
	}

	// With API key - should succeed
	resp = client.Get("/protected", rivex.WithHeader("X-API-Key", "my-secret-key"))
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 with API key, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	resp.JSON(&body)
	if body["key"] != "my-secret-key" {
		t.Errorf("expected key=my-secret-key, got %v", body["key"])
	}
}

func TestHTTPBearer(t *testing.T) {
	bearer := rivex.NewHTTPBearer()
	app := rivex.New()

	app.Get("/protected", func(ctx *rivex.Context) error {
		creds, err := bearer.Resolve(ctx)
		if err != nil {
			return err
		}
		c := creds.(*rivex.HTTPAuthorizationCredentials)
		return ctx.SendJSON(map[string]string{
			"scheme": c.Scheme,
			"token":  c.Credentials,
		})
	})

	client := rivex.NewTestClient(app)

	// Without auth - should fail
	resp := client.Get("/protected")
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403 without auth, got %d", resp.StatusCode)
	}

	// With bearer token - should succeed
	resp = client.Get("/protected", rivex.WithHeader("Authorization", "Bearer my-token"))
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 with bearer, got %d: %s", resp.StatusCode, resp.Text())
	}

	var body map[string]string
	resp.JSON(&body)
	if body["scheme"] != "Bearer" {
		t.Errorf("expected scheme=Bearer, got %s", body["scheme"])
	}
	if body["token"] != "my-token" {
		t.Errorf("expected token=my-token, got %s", body["token"])
	}
}

func TestOAuth2PasswordBearer(t *testing.T) {
	oauth2 := rivex.NewOAuth2PasswordBearer("/token")
	app := rivex.New()

	app.Get("/protected", func(ctx *rivex.Context) error {
		token, err := oauth2.Resolve(ctx)
		if err != nil {
			return err
		}
		return ctx.SendJSON(map[string]interface{}{"token": token})
	})

	client := rivex.NewTestClient(app)

	// Without auth - should fail with 401
	resp := client.Get("/protected")
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}

	// With bearer token
	resp = client.Get("/protected", rivex.WithHeader("Authorization", "Bearer access-token"))
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// --- Dependency Tests ---

func TestDependencyFunc(t *testing.T) {
	dep := rivex.Depends(func(ctx *rivex.Context) (interface{}, error) {
		return "resolved-value", nil
	})

	app := rivex.New()
	app.Get("/dep", func(ctx *rivex.Context) error {
		val, err := dep(ctx)
		if err != nil {
			return err
		}
		return ctx.SendJSON(map[string]interface{}{"value": val})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/dep")

	var body map[string]interface{}
	resp.JSON(&body)
	if body["value"] != "resolved-value" {
		t.Errorf("expected value=resolved-value, got %v", body["value"])
	}
}

func TestDependencyMiddleware(t *testing.T) {
	dep := rivex.Depends(func(ctx *rivex.Context) (interface{}, error) {
		return "injected", nil
	})

	app := rivex.New()
	app.Use(rivex.DependencyMiddleware("dep_value", dep))

	app.Get("/test", func(ctx *rivex.Context) error {
		val, _ := ctx.Get("dep_value")
		return ctx.SendJSON(map[string]interface{}{"dep": val})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/test")

	var body map[string]interface{}
	resp.JSON(&body)
	if body["dep"] != "injected" {
		t.Errorf("expected dep=injected, got %v", body["dep"])
	}
}

// --- Background Tasks Tests ---

func TestBackgroundTask(t *testing.T) {
	executed := false
	task := rivex.NewBackgroundTask(func() {
		executed = true
	})
	task.Run()
	if !executed {
		t.Error("expected background task to execute")
	}
}

func TestBackgroundTasks(t *testing.T) {
	counter := 0
	tasks := rivex.NewBackgroundTasks()
	tasks.AddTask(func() { counter++ })
	tasks.AddTask(func() { counter++ })
	tasks.AddTask(func() { counter++ })

	if tasks.Len() != 3 {
		t.Fatalf("expected 3 tasks, got %d", tasks.Len())
	}

	tasks.RunAll()
	if counter != 3 {
		t.Errorf("expected counter=3, got %d", counter)
	}
}

func TestBackgroundTaskPanicRecovery(t *testing.T) {
	task := rivex.NewBackgroundTask(func() {
		panic("test panic")
	})
	// Should not panic the test
	task.Run()
}

// --- Status Tests ---

func TestStatusCodes(t *testing.T) {
	tests := []struct {
		code int
		text string
	}{
		{200, "OK"},
		{201, "Created"},
		{204, "No Content"},
		{301, "Moved Permanently"},
		{302, "Found"},
		{400, "Bad Request"},
		{401, "Unauthorized"},
		{403, "Forbidden"},
		{404, "Not Found"},
		{405, "Method Not Allowed"},
		{409, "Conflict"},
		{422, "Unprocessable Entity"},
		{429, "Too Many Requests"},
		{500, "Internal Server Error"},
		{502, "Bad Gateway"},
		{503, "Service Unavailable"},
	}

	for _, tt := range tests {
		got := rivex.StatusText(tt.code)
		if got != tt.text {
			t.Errorf("StatusText(%d) = %q, want %q", tt.code, got, tt.text)
		}
	}
}

func TestStatusConstants(t *testing.T) {
	if rivex.StatusOK != 200 {
		t.Error("StatusOK should be 200")
	}
	if rivex.StatusCreated != 201 {
		t.Error("StatusCreated should be 201")
	}
	if rivex.StatusNotFound != 404 {
		t.Error("StatusNotFound should be 404")
	}
}

// --- OpenAPI Tests ---

func TestGenerateOpenAPI(t *testing.T) {
	app := rivex.New(
		rivex.WithTitle("Test API"),
		rivex.WithVersion("2.0.0"),
		rivex.WithDescription("A test API"),
	)

	app.Get("/users", func(ctx *rivex.Context) error { return nil }, rivex.WithTags("users"))
	app.Get("/users/{id}", func(ctx *rivex.Context) error { return nil }, rivex.WithTags("users"))
	app.Post("/users", func(ctx *rivex.Context) error { return nil },
		rivex.WithTags("users"),
		rivex.WithStatusCode(201),
		rivex.WithSummary("Create a user"),
	)

	spec := rivex.GenerateOpenAPI(app)

	if spec["openapi"] != "3.1.0" {
		t.Errorf("expected openapi=3.1.0, got %v", spec["openapi"])
	}

	info := spec["info"].(map[string]interface{})
	if info["title"] != "Test API" {
		t.Errorf("expected title=Test API, got %v", info["title"])
	}
	if info["version"] != "2.0.0" {
		t.Errorf("expected version=2.0.0, got %v", info["version"])
	}

	paths := spec["paths"].(map[string]interface{})
	if _, ok := paths["/users"]; !ok {
		t.Error("expected /users path in spec")
	}
	if _, ok := paths["/users/{id}"]; !ok {
		t.Error("expected /users/{id} path in spec")
	}

	// Check that path parameters are extracted
	userByID := paths["/users/{id}"].(map[string]interface{})
	getOp := userByID["get"].(map[string]interface{})
	params := getOp["parameters"].([]map[string]interface{})
	if len(params) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(params))
	}
	if params[0]["name"] != "id" {
		t.Errorf("expected param name=id, got %v", params[0]["name"])
	}
}

// --- Upload Tests ---

func TestUploadFile(t *testing.T) {
	uf := rivex.NewUploadFile("test.txt", "text/plain", []byte("hello world"))

	if uf.Filename != "test.txt" {
		t.Errorf("expected filename=test.txt, got %s", uf.Filename)
	}
	if uf.ContentType != "text/plain" {
		t.Errorf("expected content-type=text/plain, got %s", uf.ContentType)
	}
	if uf.Size != 11 {
		t.Errorf("expected size=11, got %d", uf.Size)
	}

	data, err := uf.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world" {
		t.Errorf("expected 'hello world', got %s", string(data))
	}
}

// --- Headers Tests ---

func TestHeaders(t *testing.T) {
	h := rivex.NewHeaders([][2]string{
		{"Content-Type", "application/json"},
		{"X-Custom", "value1"},
		{"x-custom", "value2"},
	})

	if h.Get("content-type") != "application/json" {
		t.Error("expected case-insensitive header lookup")
	}

	all := h.GetAll("x-custom")
	if len(all) != 2 {
		t.Errorf("expected 2 values for x-custom, got %d", len(all))
	}

	if !h.Has("Content-Type") {
		t.Error("expected Has to return true for Content-Type")
	}
	if h.Has("Nonexistent") {
		t.Error("expected Has to return false for Nonexistent")
	}

	if h.Len() != 3 {
		t.Errorf("expected Len=3, got %d", h.Len())
	}
}

// --- Integration test: full CRUD flow ---

func TestTodoCRUDFlow(t *testing.T) {
	app := rivex.New()

	type Todo struct {
		ID        int    `json:"id"`
		Title     string `json:"title"`
		Completed bool   `json:"completed"`
	}

	todos := make(map[int]*Todo)
	nextID := 1

	app.Get("/todos", func(ctx *rivex.Context) error {
		list := make([]*Todo, 0, len(todos))
		for _, t := range todos {
			list = append(list, t)
		}
		return ctx.SendJSON(list)
	})

	app.Post("/todos", func(ctx *rivex.Context) error {
		var input struct {
			Title string `json:"title"`
		}
		if err := ctx.JSON(&input); err != nil {
			return rivex.NewHTTPException(400, "Invalid body")
		}
		todo := &Todo{ID: nextID, Title: input.Title}
		todos[nextID] = todo
		nextID++
		return ctx.SendJSONWithStatus(todo, 201)
	}, rivex.WithStatusCode(201))

	app.Get("/todos/{id}", func(ctx *rivex.Context) error {
		id := 0
		for _, ch := range ctx.PathParam("id") {
			id = id*10 + int(ch-'0')
		}
		todo, ok := todos[id]
		if !ok {
			return rivex.NewHTTPException(404, "Not found")
		}
		return ctx.SendJSON(todo)
	})

	app.Delete("/todos/{id}", func(ctx *rivex.Context) error {
		id := 0
		for _, ch := range ctx.PathParam("id") {
			id = id*10 + int(ch-'0')
		}
		if _, ok := todos[id]; !ok {
			return rivex.NewHTTPException(404, "Not found")
		}
		delete(todos, id)
		return ctx.SendJSONWithStatus(nil, 204)
	})

	client := rivex.NewTestClient(app)

	// Create
	resp := client.Post("/todos", rivex.WithJSONBody(map[string]string{"title": "Buy milk"}))
	if resp.StatusCode != 201 {
		t.Fatalf("create: expected 201, got %d: %s", resp.StatusCode, resp.Text())
	}

	var created Todo
	resp.JSON(&created)
	if created.Title != "Buy milk" {
		t.Errorf("create: expected title=Buy milk, got %s", created.Title)
	}

	// Read
	resp = client.Get("/todos/1")
	if resp.StatusCode != 200 {
		t.Fatalf("read: expected 200, got %d", resp.StatusCode)
	}

	// List
	resp = client.Get("/todos")
	if resp.StatusCode != 200 {
		t.Fatalf("list: expected 200, got %d", resp.StatusCode)
	}

	var list []Todo
	resp.JSON(&list)
	if len(list) != 1 {
		t.Errorf("list: expected 1 todo, got %d", len(list))
	}

	// Delete
	resp = client.Delete("/todos/1")
	if resp.StatusCode != 204 {
		t.Fatalf("delete: expected 204, got %d", resp.StatusCode)
	}

	// Confirm deletion
	resp = client.Get("/todos/1")
	if resp.StatusCode != 404 {
		t.Fatalf("after delete: expected 404, got %d", resp.StatusCode)
	}
}
