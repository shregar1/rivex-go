package rivex

import (
	"strings"
)

// RouteOption configures an individual route registration.
type RouteOption func(*routeConfig)

type routeConfig struct {
	statusCode      int
	tags            []string
	summary         string
	description     string
	operationID     string
	deprecated      bool
	includeInSchema bool
}

func defaultRouteConfig() routeConfig {
	return routeConfig{
		statusCode:      StatusOK,
		includeInSchema: true,
	}
}

// WithStatusCode sets the default response status code for the route.
func WithStatusCode(code int) RouteOption {
	return func(c *routeConfig) {
		c.statusCode = code
	}
}

// WithTags sets the OpenAPI tags for the route.
func WithTags(tags ...string) RouteOption {
	return func(c *routeConfig) {
		c.tags = tags
	}
}

// WithSummary sets the OpenAPI summary for the route.
func WithSummary(summary string) RouteOption {
	return func(c *routeConfig) {
		c.summary = summary
	}
}

// WithRouteDescription sets the OpenAPI description for the route.
func WithRouteDescription(desc string) RouteOption {
	return func(c *routeConfig) {
		c.description = desc
	}
}

// WithOperationID sets the OpenAPI operation ID for the route.
func WithOperationID(id string) RouteOption {
	return func(c *routeConfig) {
		c.operationID = id
	}
}

// WithDeprecated marks the route as deprecated in the OpenAPI schema.
func WithDeprecated(deprecated bool) RouteOption {
	return func(c *routeConfig) {
		c.deprecated = deprecated
	}
}

// WithIncludeInSchema controls whether the route appears in the OpenAPI schema.
func WithIncludeInSchema(include bool) RouteOption {
	return func(c *routeConfig) {
		c.includeInSchema = include
	}
}

// RouteRecord stores the metadata for a single registered route.
type RouteRecord struct {
	Method          string
	Path            string
	Handler         HandlerFunc
	StatusCode      int
	Tags            []string
	Summary         string
	Description     string
	OperationID     string
	Deprecated      bool
	IncludeInSchema bool
}

// RouterOption configures router inclusion behavior.
type RouterOption func(*routerConfig)

type routerConfig struct {
	prefix string
	tags   []string
}

// WithPrefix sets a path prefix when including a router.
func WithPrefix(prefix string) RouterOption {
	return func(c *routerConfig) {
		c.prefix = prefix
	}
}

// WithRouterTags adds tags when including a router.
func WithRouterTags(tags ...string) RouterOption {
	return func(c *routerConfig) {
		c.tags = tags
	}
}

// Router groups related routes under a common prefix and set of tags.
type Router struct {
	prefix string
	tags   []string
	routes []RouteRecord
}

// NewRouter creates a new Router with the given options.
func NewRouter(opts ...RouterOption) *Router {
	cfg := &routerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return &Router{
		prefix: cfg.prefix,
		tags:   cfg.tags,
	}
}

// Routes returns all route records registered on this router.
func (r *Router) Routes() []RouteRecord {
	cp := make([]RouteRecord, len(r.routes))
	copy(cp, r.routes)
	return cp
}

// Get registers a GET route on the router.
func (r *Router) Get(path string, handler HandlerFunc, opts ...RouteOption) *Router {
	r.addRoute("GET", path, handler, opts...)
	return r
}

// Post registers a POST route on the router.
func (r *Router) Post(path string, handler HandlerFunc, opts ...RouteOption) *Router {
	r.addRoute("POST", path, handler, opts...)
	return r
}

// Put registers a PUT route on the router.
func (r *Router) Put(path string, handler HandlerFunc, opts ...RouteOption) *Router {
	r.addRoute("PUT", path, handler, opts...)
	return r
}

// Delete registers a DELETE route on the router.
func (r *Router) Delete(path string, handler HandlerFunc, opts ...RouteOption) *Router {
	r.addRoute("DELETE", path, handler, opts...)
	return r
}

// Patch registers a PATCH route on the router.
func (r *Router) Patch(path string, handler HandlerFunc, opts ...RouteOption) *Router {
	r.addRoute("PATCH", path, handler, opts...)
	return r
}

// Head registers a HEAD route on the router.
func (r *Router) Head(path string, handler HandlerFunc, opts ...RouteOption) *Router {
	r.addRoute("HEAD", path, handler, opts...)
	return r
}

// Options registers an OPTIONS route on the router.
func (r *Router) Options(path string, handler HandlerFunc, opts ...RouteOption) *Router {
	r.addRoute("OPTIONS", path, handler, opts...)
	return r
}

// IncludeRouter merges another router's routes into this router.
func (r *Router) IncludeRouter(other *Router, opts ...RouterOption) *Router {
	cfg := &routerConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	for _, route := range other.routes {
		merged := RouteRecord{
			Method:          route.Method,
			Path:            joinPaths(r.prefix, cfg.prefix, route.Path),
			Handler:         route.Handler,
			StatusCode:      route.StatusCode,
			Tags:            mergeTags(r.tags, cfg.tags, route.Tags),
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

// IncludeController merges a Controller's routes into this router.
func (r *Router) IncludeController(ctrl Controller, opts ...RouterOption) *Router {
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
			Path:            joinPaths(r.prefix, cfg.prefix, ctrlPath, route.Path),
			Handler:         route.Handler,
			StatusCode:      route.StatusCode,
			Tags:            mergeTags(r.tags, cfg.tags, ctrlTags, route.Tags),
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

func (r *Router) addRoute(method, path string, handler HandlerFunc, opts ...RouteOption) {
	cfg := defaultRouteConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	fullPath := joinPaths(r.prefix, path)
	tags := mergeTags(r.tags, cfg.tags)

	record := RouteRecord{
		Method:          method,
		Path:            fullPath,
		Handler:         handler,
		StatusCode:      cfg.statusCode,
		Tags:            tags,
		Summary:         cfg.summary,
		Description:     cfg.description,
		OperationID:     cfg.operationID,
		Deprecated:      cfg.deprecated,
		IncludeInSchema: cfg.includeInSchema,
	}
	r.routes = append(r.routes, record)
}

// Controller is an interface for class-style route grouping. Implementations
// provide a path prefix, tags, and a list of route records.
type Controller interface {
	// Path returns the base path prefix for all routes in this controller.
	Path() string

	// Tags returns the OpenAPI tags for all routes in this controller.
	Tags() []string

	// Routes returns the route definitions for this controller.
	Routes() []RouteRecord
}

// Route is a convenience function for building RouteRecord values inside
// Controller implementations.
func Route(method, path string, handler HandlerFunc, opts ...RouteOption) RouteRecord {
	cfg := defaultRouteConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return RouteRecord{
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
}

// joinPaths joins path segments, normalizing slashes.
func joinPaths(parts ...string) string {
	var cleaned []string
	for _, part := range parts {
		part = strings.Trim(part, "/")
		if part != "" {
			cleaned = append(cleaned, part)
		}
	}
	if len(cleaned) == 0 {
		return "/"
	}
	return "/" + strings.Join(cleaned, "/")
}

// mergeTags combines multiple tag slices, deduplicating while preserving order.
func mergeTags(slices ...[]string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, s := range slices {
		for _, tag := range s {
			if _, ok := seen[tag]; !ok {
				seen[tag] = struct{}{}
				result = append(result, tag)
			}
		}
	}
	return result
}
