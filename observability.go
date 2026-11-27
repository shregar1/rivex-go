package rivex

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type LogEntry struct {
	Timestamp  string        `json:"timestamp"`
	Method     string        `json:"method"`
	Path       string        `json:"path"`
	StatusCode int           `json:"status_code"`
	Duration   time.Duration `json:"duration"`
	ClientIP   string        `json:"client_ip"`
	RequestID  string        `json:"request_id,omitempty"`
	UserAgent  string        `json:"user_agent,omitempty"`
	Error      string        `json:"error,omitempty"`
}

type LogFormatter func(entry *LogEntry) string

type LoggingMiddleware struct {
	logger     *log.Logger
	formatter  LogFormatter
	skipPaths  []string
}

type LoggingOption func(*LoggingMiddleware)

func WithLogger(logger *log.Logger) LoggingOption {
	return func(l *LoggingMiddleware) {
		l.logger = logger
	}
}

func WithLogFormatter(formatter LogFormatter) LoggingOption {
	return func(l *LoggingMiddleware) {
		l.formatter = formatter
	}
}

func WithSkipPaths(paths ...string) LoggingOption {
	return func(l *LoggingMiddleware) {
		l.skipPaths = paths
	}
}

func NewLoggingMiddleware(opts ...LoggingOption) *LoggingMiddleware {
	l := &LoggingMiddleware{
		logger: log.Default(),
		formatter: func(e *LogEntry) string {
			b, _ := json.Marshal(e)
			return string(b)
		},
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

func (l *LoggingMiddleware) Process(ctx *Context, next NextFunc) error {
	path := ctx.Path()
	for _, skip := range l.skipPaths {
		if path == skip {
			return next(ctx)
		}
	}
	start := time.Now()
	err := next(ctx)
	duration := time.Since(start)
	entry := &LogEntry{
		Timestamp: start.UTC().Format(time.RFC3339Nano),
		Method:    ctx.Method(),
		Path:      path,
		ClientIP:  ctx.ClientHost(),
		UserAgent: ctx.Header("User-Agent"),
		RequestID: ctx.GetString("request_id"),
		Duration:  duration,
	}
	if err != nil {
		entry.Error = err.Error()
		if httpErr, ok := err.(*HTTPException); ok {
			entry.StatusCode = httpErr.StatusCode
		} else {
			entry.StatusCode = StatusInternalServerError
		}
	} else if ctx.response != nil {
		entry.StatusCode = ctx.response.StatusCode
	}
	l.logger.Println(l.formatter(entry))
	return err
}

func JSONLogFormatter(entry *LogEntry) string {
	b, _ := json.Marshal(entry)
	return string(b)
}

func TextLogFormatter(entry *LogEntry) string {
	return fmt.Sprintf("%s %s %s %d %v", entry.Method, entry.Path, entry.ClientIP, entry.StatusCode, entry.Duration)
}

type HealthCheck struct {
	Status    string            `json:"status"`
	Version   string            `json:"version,omitempty"`
	Checks    map[string]string `json:"checks,omitempty"`
	Uptime    string            `json:"uptime,omitempty"`
}

type HealthCheckFunc func() (string, error)

type HealthCheckMiddleware struct {
	version    string
	startTime  time.Time
	checks     map[string]HealthCheckFunc
	healthPath string
}

type HealthCheckOption func(*HealthCheckMiddleware)

func WithHealthVersion(version string) HealthCheckOption {
	return func(h *HealthCheckMiddleware) {
		h.version = version
	}
}

func WithHealthCheck(name string, fn HealthCheckFunc) HealthCheckOption {
	return func(h *HealthCheckMiddleware) {
		h.checks[name] = fn
	}
}

func WithHealthPath(path string) HealthCheckOption {
	return func(h *HealthCheckMiddleware) {
		h.healthPath = path
	}
}

func NewHealthCheck(opts ...HealthCheckOption) *HealthCheckMiddleware {
	h := &HealthCheckMiddleware{
		startTime:  time.Now(),
		healthPath: "/health",
		checks:     make(map[string]HealthCheckFunc),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func (h *HealthCheckMiddleware) Process(ctx *Context, next NextFunc) error {
	if ctx.Path() != h.healthPath {
		return next(ctx)
	}
	if ctx.Method() != "GET" {
		return NewHTTPException(StatusMethodNotAllowed, "Method not allowed")
	}
	status := "healthy"
	checkResults := make(map[string]string)
	for name, fn := range h.checks {
		checkStatus, err := fn()
		if err != nil {
			checkResults[name] = fmt.Sprintf("unhealthy: %v", err)
			status = "degraded"
		} else {
			checkResults[name] = checkStatus
		}
	}
	hc := HealthCheck{
		Status:  status,
		Version: h.version,
		Checks:  checkResults,
		Uptime:  time.Since(h.startTime).Round(time.Second).String(),
	}
	code := StatusOK
	if status == "degraded" {
		code = StatusServiceUnavailable
	}
	ctx.response = JSONResponse(hc, code)
	return nil
}

type RequestCounter struct {
	count   int64
	bytesIn  int64
	bytesOut int64
	errors  int64
	statusCounts map[int]int64
	mu       sync.RWMutex
}

func newRequestCounter() *RequestCounter {
	return &RequestCounter{
		statusCounts: make(map[int]int64),
	}
}

type MetricsMiddleware struct {
	counter    *RequestCounter
	skipPaths  []string
}

type MetricsOption func(*MetricsMiddleware)

func WithMetricsSkipPaths(paths ...string) MetricsOption {
	return func(m *MetricsMiddleware) {
		m.skipPaths = append(m.skipPaths, paths...)
	}
}

func NewMetricsMiddleware(opts ...MetricsOption) *MetricsMiddleware {
	m := &MetricsMiddleware{
		counter: newRequestCounter(),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *MetricsMiddleware) Process(ctx *Context, next NextFunc) error {
	path := ctx.Path()
	for _, skip := range m.skipPaths {
		if path == skip {
			return next(ctx)
		}
	}
	atomic.AddInt64(&m.counter.count, 1)
	atomic.AddInt64(&m.counter.bytesIn, int64(len(ctx.Body())))
	err := next(ctx)
	if err != nil {
		atomic.AddInt64(&m.counter.errors, 1)
		if httpErr, ok := err.(*HTTPException); ok {
			m.recordStatus(httpErr.StatusCode)
		} else {
			m.recordStatus(StatusInternalServerError)
		}
		return err
	}
	if ctx.response != nil {
		atomic.AddInt64(&m.counter.bytesOut, int64(len(ctx.response.Content)))
		m.recordStatus(ctx.response.StatusCode)
	}
	return nil
}

func (m *MetricsMiddleware) recordStatus(code int) {
	m.counter.mu.Lock()
	m.counter.statusCounts[code]++
	m.counter.mu.Unlock()
}

func (m *MetricsMiddleware) Snapshot() map[string]interface{} {
	m.counter.mu.RLock()
	defer m.counter.mu.RUnlock()
	statuses := make(map[string]int64)
	for code, count := range m.counter.statusCounts {
		statuses[strconv.Itoa(code)] = count
	}
	return map[string]interface{}{
		"total_requests":  atomic.LoadInt64(&m.counter.count),
		"total_bytes_in":  atomic.LoadInt64(&m.counter.bytesIn),
		"total_bytes_out": atomic.LoadInt64(&m.counter.bytesOut),
		"total_errors":    atomic.LoadInt64(&m.counter.errors),
		"status_codes":    statuses,
	}
}

func (m *MetricsMiddleware) MetricsHandler(ctx *Context) error {
	ctx.response = JSONResponse(m.Snapshot())
	return nil
}

type MiddlewareChain []Middleware

func Chain(middleware ...Middleware) Middleware {
	chain := MiddlewareChain(middleware)
	return &chain
}

func (c MiddlewareChain) Process(ctx *Context, next NextFunc) error {
	if len(c) == 0 {
		return next(ctx)
	}
	return c[0].Process(ctx, func(ctx *Context) error {
		sub := MiddlewareChain(c[1:])
		return sub.Process(ctx, next)
	})
}

type RateLimitMiddleware struct {
	limiter    *RateLimiter
	keyFunc    func(ctx *Context) string
	autoError  bool
}

type RateLimitOption func(*RateLimitMiddleware)

func WithRateLimitKeyFunc(fn func(ctx *Context) string) RateLimitOption {
	return func(r *RateLimitMiddleware) {
		r.keyFunc = fn
	}
}

func WithRateLimitAutoError(autoError bool) RateLimitOption {
	return func(r *RateLimitMiddleware) {
		r.autoError = autoError
	}
}

func NewRateLimitMiddleware(rate float64, burst uint32, opts ...RateLimitOption) *RateLimitMiddleware {
	r := &RateLimitMiddleware{
		limiter:   NewRateLimiter(rate, burst),
		keyFunc:   func(ctx *Context) string { return ctx.ClientHost() },
		autoError: true,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (r *RateLimitMiddleware) Process(ctx *Context, next NextFunc) error {
	key := r.keyFunc(ctx)
	if !r.limiter.Check(key) {
		if r.autoError {
			headers := map[string]string{
				"Retry-After":     "1",
				"X-RateLimit-Limit":    "1",
			}
			return NewHTTPException(StatusTooManyRequests, "Rate limit exceeded").WithHeaders(headers)
		}
		return next(ctx)
	}
	return next(ctx)
}

func IPRateLimitKey(ctx *Context) string {
	ip := ctx.ClientHost()
	if ip == "" {
		return "unknown"
	}
	return ip
}

type CORPSCheckMiddleware struct{}

func NewCORPSCheck() *CORPSCheckMiddleware {
	return &CORPSCheckMiddleware{}
}

func (c *CORPSCheckMiddleware) Process(ctx *Context, next NextFunc) error {
	origin := ctx.Header("Origin")
	err := next(ctx)
	if err != nil {
		return err
	}
	if origin != "" && ctx.response != nil {
		if ctx.response.Headers["Access-Control-Allow-Origin"] == "" {
			ctx.response.Headers["Access-Control-Allow-Origin"] = origin
		}
	}
	return nil
}

type CORPSSupport struct {
	allowedOrigins map[string]bool
	allowAll       bool
}

func NewCORPSSupport(origins ...string) *CORPSSupport {
	c := &CORPSSupport{
		allowedOrigins: make(map[string]bool),
	}
	for _, o := range origins {
		if o == "*" {
			c.allowAll = true
			break
		}
		c.allowedOrigins[o] = true
	}
	return c
}

func (c *CORPSSupport) IsAllowed(origin string) bool {
	if c.allowAll {
		return true
	}
	if c.allowedOrigins[origin] {
		return true
	}
	if strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "https://localhost") {
		return true
	}
	return false
}
