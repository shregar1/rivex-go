// Package rivex provides a high-performance web framework for Go, backed by
// a Rust HTTP runtime. It offers a clean, idiomatic Go API with support for
// routing, middleware, dependency injection, OpenAPI generation, and more.
//
// Basic usage:
//
//	app := rivex.New(rivex.WithTitle("My API"))
//
//	app.Get("/", func(ctx *rivex.Context) error {
//	    return ctx.SendJSON(map[string]string{"message": "Hello!"})
//	})
//
//	app.Run("127.0.0.1", 8000)
package rivex

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/rivex-dev/rivex-go/internal/ffi"
)

// HandlerFunc is the signature for request handlers. Handlers receive a Context
// and return an error. To send a response, use the Context's Send* methods or
// set a Response directly.
type HandlerFunc func(ctx *Context) error

// ExceptionHandlerFunc handles errors that occur during request processing.
type ExceptionHandlerFunc func(ctx *Context, err error) error

// LifespanFunc is called when the application starts and should return a
// cleanup function that is called when the application shuts down.
type LifespanFunc func(app *Rivex) func()

// Option configures the Rivex application.
type Option func(*Rivex)

// WithTitle sets the API title used in OpenAPI documentation.
func WithTitle(title string) Option {
	return func(r *Rivex) {
		r.title = title
	}
}

// WithVersion sets the API version used in OpenAPI documentation.
func WithVersion(version string) Option {
	return func(r *Rivex) {
		r.version = version
	}
}

// WithDescription sets the API description used in OpenAPI documentation.
func WithDescription(desc string) Option {
	return func(r *Rivex) {
		r.description = desc
	}
}

// WithDebug enables debug mode, which includes stack traces in error responses.
func WithDebug(debug bool) Option {
	return func(r *Rivex) {
		r.debug = debug
	}
}

// WithDocsURL sets the URL path for the interactive API documentation page.
// Set to empty string to disable the docs endpoint.
func WithDocsURL(url string) Option {
	return func(r *Rivex) {
		r.docsURL = url
	}
}

// WithOpenAPIURL sets the URL path for the OpenAPI JSON specification.
// Set to empty string to disable the OpenAPI endpoint.
func WithOpenAPIURL(url string) Option {
	return func(r *Rivex) {
		r.openAPIURL = url
	}
}

// WithLifespan sets a lifespan function that is called on startup and returns
// a cleanup function called on shutdown.
func WithLifespan(fn LifespanFunc) Option {
	return func(r *Rivex) {
		r.lifespan = fn
	}
}

// WithOpenAPITags sets custom OpenAPI tag definitions.
func WithOpenAPITags(tags []map[string]interface{}) Option {
	return func(r *Rivex) {
		r.openAPITags = tags
	}
}

// WithServers sets the OpenAPI server definitions.
func WithServers(servers []map[string]interface{}) Option {
	return func(r *Rivex) {
		r.servers = servers
	}
}

// Rivex is the main application type. It manages routes, middleware,
// exception handlers, and lifecycle hooks.
type Rivex struct {
	title       string
	version     string
	description string
	debug       bool
	docsURL     string
	openAPIURL  string
	lifespan    LifespanFunc
	openAPITags []map[string]interface{}
	servers     []map[string]interface{}

	routes            []RouteRecord
	middleware        []Middleware
	startupHooks      []func()
	shutdownHooks     []func()
	exceptionHandlers map[string]ExceptionHandlerFunc

	// State is an arbitrary key-value store for application-level state that
	// persists for the lifetime of the application.
	State map[string]interface{}
}

// New creates a new Rivex application with the given options.
func New(opts ...Option) *Rivex {
	r := &Rivex{
		title:             "Rivex",
		version:           "0.1.0",
		docsURL:           "/docs",
		openAPIURL:        "/openapi.json",
		exceptionHandlers: make(map[string]ExceptionHandlerFunc),
		State:             make(map[string]interface{}),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Get registers a handler for GET requests at the given path.
func (r *Rivex) Get(path string, handler HandlerFunc, opts ...RouteOption) *Rivex {
	r.addRoute("GET", path, handler, opts...)
	return r
}

// Post registers a handler for POST requests at the given path.
func (r *Rivex) Post(path string, handler HandlerFunc, opts ...RouteOption) *Rivex {
	r.addRoute("POST", path, handler, opts...)
	return r
}

// Put registers a handler for PUT requests at the given path.
func (r *Rivex) Put(path string, handler HandlerFunc, opts ...RouteOption) *Rivex {
	r.addRoute("PUT", path, handler, opts...)
	return r
}

// Delete registers a handler for DELETE requests at the given path.
func (r *Rivex) Delete(path string, handler HandlerFunc, opts ...RouteOption) *Rivex {
	r.addRoute("DELETE", path, handler, opts...)
	return r
}

// Patch registers a handler for PATCH requests at the given path.
func (r *Rivex) Patch(path string, handler HandlerFunc, opts ...RouteOption) *Rivex {
	r.addRoute("PATCH", path, handler, opts...)
	return r
}

// Head registers a handler for HEAD requests at the given path.
func (r *Rivex) Head(path string, handler HandlerFunc, opts ...RouteOption) *Rivex {
	r.addRoute("HEAD", path, handler, opts...)
	return r
}

// Options registers a handler for OPTIONS requests at the given path.
func (r *Rivex) Options(path string, handler HandlerFunc, opts ...RouteOption) *Rivex {
	r.addRoute("OPTIONS", path, handler, opts...)
	return r
}

// IncludeRouter merges all routes from the given router into the application,
// optionally applying a prefix and additional tags.
func (r *Rivex) IncludeRouter(router *Router, opts ...RouterOption) *Rivex {
	cfg := &routerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	for _, route := range router.routes {
		merged := RouteRecord{
			Method:          route.Method,
			Path:            joinPaths(cfg.prefix, route.Path),
			Handler:         route.Handler,
			StatusCode:      route.StatusCode,
			Tags:            mergeTags(cfg.tags, route.Tags),
			Summary:         route.Summary,
			Description:     route.Description,
			OperationID:     route.OperationID,
			Deprecated:      route.Deprecated,
			IncludeInSchema: route.IncludeInSchema,
		}
		r.routes = append(r.routes, merged)
	}
	return r
}

// IncludeController merges all routes from the given Controller into the
// application, optionally applying a prefix and additional tags.
func (r *Rivex) IncludeController(ctrl Controller, opts ...RouterOption) *Rivex {
	cfg := &routerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	ctrlRoutes := ctrl.Routes()
	ctrlPath := ctrl.Path()
	ctrlTags := ctrl.Tags()

	for _, route := range ctrlRoutes {
		merged := RouteRecord{
			Method:          route.Method,
			Path:            joinPaths(cfg.prefix, ctrlPath, route.Path),
			Handler:         route.Handler,
			StatusCode:      route.StatusCode,
			Tags:            mergeTags(cfg.tags, ctrlTags, route.Tags),
			Summary:         route.Summary,
			Description:     route.Description,
			OperationID:     route.OperationID,
			Deprecated:      route.Deprecated,
			IncludeInSchema: route.IncludeInSchema,
		}
		r.routes = append(r.routes, merged)
	}
	return r
}

// Use adds one or more middleware to the application's middleware chain.
// Middleware are executed in the order they are added.
func (r *Rivex) Use(middleware ...Middleware) *Rivex {
	r.middleware = append(r.middleware, middleware...)
	return r
}

// OnEvent registers a lifecycle hook. Supported events are "startup" and
// "shutdown".
func (r *Rivex) OnEvent(event string, fn func()) *Rivex {
	switch event {
	case "startup":
		r.startupHooks = append(r.startupHooks, fn)
	case "shutdown":
		r.shutdownHooks = append(r.shutdownHooks, fn)
	}
	return r
}

// ExceptionHandler registers a custom handler for errors of the given type.
// The typeName should be the string representation of the error type.
func (r *Rivex) ExceptionHandler(typeName string, handler ExceptionHandlerFunc) *Rivex {
	r.exceptionHandlers[typeName] = handler
	return r
}

// GetRoutes returns a copy of all registered route records.
func (r *Rivex) GetRoutes() []RouteRecord {
	cp := make([]RouteRecord, len(r.routes))
	copy(cp, r.routes)
	return cp
}

// Run starts the HTTP server on the given host and port. This method blocks
// until the server is shut down.
func (r *Rivex) Run(host string, port int) error {
	// Run startup hooks
	var cleanupFn func()
	if r.lifespan != nil {
		cleanupFn = r.lifespan(r)
	}
	for _, hook := range r.startupHooks {
		hook()
	}

	// Register routes with the FFI server
	server := ffi.ServerNew()

	// Register OpenAPI routes if configured
	if r.openAPIURL != "" {
		r.registerOpenAPIRoutes()
	}

	for _, route := range r.routes {
		handler := r.buildHandler(route)
		ffi.ServerAddRoute(server, route.Method, route.Path, handler)
	}

	log.Printf("Rivex server starting on %s:%d", host, port)

	// Start the server (blocks until shutdown)
	err := ffi.ServerRun(server, host, port)

	// Run shutdown hooks
	for _, hook := range r.shutdownHooks {
		hook()
	}
	if cleanupFn != nil {
		cleanupFn()
	}

	return err
}

// addRoute is the internal route registration method.
func (r *Rivex) addRoute(method, path string, handler HandlerFunc, opts ...RouteOption) {
	cfg := defaultRouteConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	record := RouteRecord{
		Method:          method,
		Path:            path,
		Handler:         handler,
		StatusCode:      cfg.statusCode,
		Tags:            cfg.tags,
		Summary:         cfg.summary,
		Description:     cfg.description,
		OperationID:     cfg.operationID,
		Deprecated:      cfg.deprecated,
		IncludeInSchema: cfg.includeInSchema,
	}
	r.routes = append(r.routes, record)
}

// buildHandler wraps a route's handler with middleware and error handling.
func (r *Rivex) buildHandler(route RouteRecord) ffi.RequestHandler {
	return func(method, path string, headers [][2]string, body []byte, queryString string, pathParams map[string]string, clientHost string, clientPort int) ffi.RawResponse {
		ctx := newContext(method, path, headers, body, queryString, pathParams, clientHost, clientPort)

		// Build the handler chain with middleware
		final := func(ctx *Context) error {
			err := route.Handler(ctx)
			if err != nil {
				return err
			}
			// If the handler didn't set a response, create a default 204
			if ctx.response == nil {
				ctx.response = NewResponse(nil, StatusNoContent, nil)
			}
			return nil
		}

		handler := r.applyMiddleware(final)

		// Execute the handler chain
		err := handler(ctx)
		if err != nil {
			return r.handleError(ctx, err, route.StatusCode)
		}

		resp := ctx.response
		if resp == nil {
			resp = NewResponse(nil, StatusNoContent, nil)
		}

		// Override status code if the route specifies one and the response is
		// using the default 200.
		if route.StatusCode != StatusOK && resp.StatusCode == StatusOK {
			resp.StatusCode = route.StatusCode
		}

		return toRawResponse(resp)
	}
}

// applyMiddleware wraps a handler with the middleware chain.
func (r *Rivex) applyMiddleware(handler HandlerFunc) HandlerFunc {
	h := handler
	for i := len(r.middleware) - 1; i >= 0; i-- {
		mw := r.middleware[i]
		next := h
		h = func(ctx *Context) error {
			return mw.Process(ctx, NextFunc(next))
		}
	}
	return h
}

// handleError converts an error into an appropriate HTTP response.
func (r *Rivex) handleError(ctx *Context, err error, defaultStatus int) ffi.RawResponse {
	// Check for custom exception handlers
	errTypeName := fmt.Sprintf("%T", err)
	if handler, ok := r.exceptionHandlers[errTypeName]; ok {
		handlerErr := handler(ctx, err)
		if handlerErr == nil && ctx.response != nil {
			return toRawResponse(ctx.response)
		}
	}

	// Handle HTTPException
	if httpErr, ok := err.(*HTTPException); ok {
		return toRawResponse(httpErr.toResponse())
	}

	// Default error response
	status := StatusInternalServerError
	detail := "Internal Server Error"
	if r.debug {
		detail = err.Error()
	}

	body, _ := json.Marshal(map[string]interface{}{"detail": detail})
	resp := NewResponse(body, status, map[string]string{"Content-Type": "application/json"})
	return toRawResponse(resp)
}

// registerOpenAPIRoutes adds the OpenAPI spec and docs endpoints.
func (r *Rivex) registerOpenAPIRoutes() {
	spec := GenerateOpenAPI(r)
	specBytes, _ := json.MarshalIndent(spec, "", "  ")

	// OpenAPI JSON endpoint
	r.addRoute("GET", r.openAPIURL, func(ctx *Context) error {
		ctx.response = NewResponse(specBytes, StatusOK, map[string]string{
			"Content-Type": "application/json",
		})
		return nil
	}, WithIncludeInSchema(false))

	// Docs HTML endpoint
	if r.docsURL != "" {
		docsHTML := generateDocsHTML(r.title, r.openAPIURL)
		r.addRoute("GET", r.docsURL, func(ctx *Context) error {
			ctx.response = NewResponse([]byte(docsHTML), StatusOK, map[string]string{
				"Content-Type": "text/html; charset=utf-8",
			})
			return nil
		}, WithIncludeInSchema(false))
	}
}

// toRawResponse converts a Response to the FFI RawResponse format.
func toRawResponse(resp *Response) ffi.RawResponse {
	var headerPairs [][2]string
	for k, v := range resp.Headers {
		headerPairs = append(headerPairs, [2]string{k, v})
	}
	return ffi.RawResponse{
		Status:  resp.StatusCode,
		Headers: headerPairs,
		Body:    resp.Content,
	}
}

// matchPath attempts to match a URL path against a route pattern containing
// {param} placeholders. Returns the extracted parameters and whether the match
// was successful.
func matchPath(pattern, path string) (map[string]string, bool) {
	if pattern == "/" {
		return map[string]string{}, path == "/"
	}

	var names []string
	var parts []string
	for _, segment := range strings.Split(strings.Trim(pattern, "/"), "/") {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			names = append(names, segment[1:len(segment)-1])
			parts = append(parts, "([^/]+)")
		} else {
			parts = append(parts, regexp.QuoteMeta(segment))
		}
	}

	regexStr := "^/" + strings.Join(parts, "/") + "$"
	re, err := regexp.Compile(regexStr)
	if err != nil {
		return nil, false
	}

	matches := re.FindStringSubmatch(path)
	if matches == nil {
		return nil, false
	}

	params := make(map[string]string, len(names))
	for i, name := range names {
		params[name] = matches[i+1]
	}
	return params, true
}
