package rivex

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// Middleware is the interface for request/response interceptors. Middleware
// implementations wrap the handler chain and can modify the context or
// short-circuit processing.
type Middleware interface {
	// Process handles the request. Call next(ctx) to continue the chain.
	Process(ctx *Context, next NextFunc) error
}

// NextFunc is the type for the next handler in the middleware chain.
type NextFunc func(ctx *Context) error

// MiddlewareFunc is an adapter that allows using ordinary functions as
// Middleware.
type MiddlewareFunc func(ctx *Context, next NextFunc) error

// Process implements the Middleware interface for MiddlewareFunc.
func (f MiddlewareFunc) Process(ctx *Context, next NextFunc) error {
	return f(ctx, next)
}

// --- CORS Middleware ---

// CORSOption configures the CORS middleware.
type CORSOption func(*CORSMiddleware)

// AllowOrigins sets the allowed origins for CORS requests.
func AllowOrigins(origins ...string) CORSOption {
	return func(c *CORSMiddleware) {
		c.allowOrigins = origins
	}
}

// AllowMethods sets the allowed HTTP methods for CORS requests.
func AllowMethods(methods ...string) CORSOption {
	return func(c *CORSMiddleware) {
		c.allowMethods = methods
	}
}

// AllowHeaders sets the allowed headers for CORS requests.
func AllowHeaders(headers ...string) CORSOption {
	return func(c *CORSMiddleware) {
		c.allowHeaders = headers
	}
}

// AllowCredentials enables credentials (cookies, authorization headers) in
// CORS requests.
func AllowCredentials(allow bool) CORSOption {
	return func(c *CORSMiddleware) {
		c.allowCredentials = allow
	}
}

// MaxAge sets the maximum age (in seconds) for CORS preflight cache.
func MaxAge(age int) CORSOption {
	return func(c *CORSMiddleware) {
		c.maxAge = age
	}
}

// CORSMiddleware handles Cross-Origin Resource Sharing (CORS) headers.
type CORSMiddleware struct {
	allowOrigins     []string
	allowMethods     []string
	allowHeaders     []string
	allowCredentials bool
	maxAge           int
}

// NewCORS creates a new CORS middleware with the given options.
func NewCORS(opts ...CORSOption) *CORSMiddleware {
	c := &CORSMiddleware{
		allowOrigins: []string{"*"},
		allowMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		allowHeaders: []string{"*"},
		maxAge:       600,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Process implements the Middleware interface for CORS.
func (c *CORSMiddleware) Process(ctx *Context, next NextFunc) error {
	origin := ctx.Header("Origin")

	// Handle preflight requests
	if ctx.Method() == "OPTIONS" {
		ctx.response = NewResponse(nil, StatusNoContent, c.corsHeaders(origin))
		return nil
	}

	// Process the request
	err := next(ctx)
	if err != nil {
		return err
	}

	// Add CORS headers to the response
	if ctx.response != nil {
		for k, v := range c.corsHeaders(origin) {
			ctx.response.Headers[k] = v
		}
	}
	return nil
}

func (c *CORSMiddleware) corsHeaders(requestOrigin string) map[string]string {
	headers := make(map[string]string)

	// Determine allowed origin
	var allowedOrigin string
	if containsString(c.allowOrigins, "*") {
		if c.allowCredentials {
			// Wildcard + credentials is invalid per spec; reflect origin instead
			if requestOrigin != "" {
				allowedOrigin = requestOrigin
			} else {
				allowedOrigin = "*"
			}
		} else {
			allowedOrigin = "*"
		}
	} else if requestOrigin != "" && containsString(c.allowOrigins, requestOrigin) {
		allowedOrigin = requestOrigin
	} else if len(c.allowOrigins) > 0 {
		allowedOrigin = c.allowOrigins[0]
	}

	headers["Access-Control-Allow-Origin"] = allowedOrigin
	headers["Access-Control-Allow-Methods"] = strings.Join(c.allowMethods, ", ")
	headers["Access-Control-Allow-Headers"] = strings.Join(c.allowHeaders, ", ")
	headers["Access-Control-Max-Age"] = intToStr(c.maxAge)

	if c.allowCredentials {
		headers["Access-Control-Allow-Credentials"] = "true"
	}
	if !containsString(c.allowOrigins, "*") {
		headers["Vary"] = "Origin"
	}

	return headers
}

// --- GZip Middleware ---

// GZipMiddleware compresses response bodies using gzip when the client accepts
// it and the response body exceeds the minimum size threshold.
type GZipMiddleware struct {
	minSize int
}

// NewGZip creates a new GZip compression middleware. The optional minSize
// parameter sets the minimum body size in bytes before compression is applied
// (defaults to 500).
func NewGZip(minSize ...int) *GZipMiddleware {
	size := 500
	if len(minSize) > 0 {
		size = minSize[0]
	}
	return &GZipMiddleware{minSize: size}
}

// Process implements the Middleware interface for gzip compression.
func (g *GZipMiddleware) Process(ctx *Context, next NextFunc) error {
	err := next(ctx)
	if err != nil {
		return err
	}

	if ctx.response == nil || len(ctx.response.Content) < g.minSize {
		return nil
	}

	// Check if the client accepts gzip
	accept := ctx.Header("Accept-Encoding")
	if !strings.Contains(strings.ToLower(accept), "gzip") {
		return nil
	}

	// Compress the response
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	_, writeErr := writer.Write(ctx.response.Content)
	if writeErr != nil {
		return nil // Silently skip compression on error
	}
	writer.Close()

	// Only use compression if it actually reduces size
	if buf.Len() >= len(ctx.response.Content) {
		return nil
	}

	ctx.response.Content = buf.Bytes()
	delete(ctx.response.Headers, "Content-Length")
	ctx.response.Headers["Content-Encoding"] = "gzip"
	ctx.response.Headers["Vary"] = "Accept-Encoding"

	return nil
}

// --- RequestID Middleware ---

// RequestIDMiddleware adds a unique request ID to each response. If the
// incoming request already has a request ID header, that value is reused.
type RequestIDMiddleware struct {
	headerName string
}

// NewRequestID creates a new RequestID middleware. The optional headerName
// parameter specifies the header to use (defaults to "X-Request-ID").
func NewRequestID(headerName ...string) *RequestIDMiddleware {
	name := "X-Request-ID"
	if len(headerName) > 0 && headerName[0] != "" {
		name = headerName[0]
	}
	return &RequestIDMiddleware{headerName: name}
}

// Process implements the Middleware interface for request ID injection.
func (m *RequestIDMiddleware) Process(ctx *Context, next NextFunc) error {
	// Check for existing request ID
	requestID := ctx.Header(m.headerName)
	if requestID == "" {
		requestID = generateRequestID()
	}

	// Store the request ID in the context for use by handlers
	ctx.Set("request_id", requestID)

	err := next(ctx)
	if err != nil {
		return err
	}

	// Add request ID to response headers
	if ctx.response != nil {
		ctx.response.Headers[m.headerName] = requestID
	}
	return nil
}

// generateRequestID creates a random hex string suitable for use as a request
// identifier.
func generateRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// --- Trusted Host Middleware ---

// TrustedHostMiddleware rejects requests whose Host header does not match the
// allowed set of hosts.
type TrustedHostMiddleware struct {
	allowedHosts map[string]struct{}
}

// NewTrustedHost creates a new TrustedHost middleware that only allows requests
// from the specified hosts. If no hosts are provided or "*" is included, all
// hosts are allowed.
func NewTrustedHost(allowedHosts ...string) *TrustedHostMiddleware {
	hosts := make(map[string]struct{}, len(allowedHosts))
	for _, h := range allowedHosts {
		hosts[strings.ToLower(h)] = struct{}{}
	}
	if len(hosts) == 0 {
		hosts["*"] = struct{}{}
	}
	return &TrustedHostMiddleware{allowedHosts: hosts}
}

// Process implements the Middleware interface for trusted host validation.
func (m *TrustedHostMiddleware) Process(ctx *Context, next NextFunc) error {
	if _, ok := m.allowedHosts["*"]; ok {
		return next(ctx)
	}

	host := ctx.Header("Host")
	// Strip port if present
	if idx := strings.IndexByte(host, ':'); idx > 0 {
		host = host[:idx]
	}
	host = strings.ToLower(host)

	if _, ok := m.allowedHosts[host]; !ok {
		body, _ := json.Marshal(map[string]string{"detail": "Invalid host header"})
		ctx.response = NewResponse(body, StatusBadRequest, map[string]string{
			"Content-Type": "application/json",
		})
		return nil
	}

	return next(ctx)
}

// --- Helpers ---

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func intToStr(n int) string {
	return fmt.Sprintf("%d", n)
}
