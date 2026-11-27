package rivex

import (
	"sync"
	"time"
)

type CachedResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
	ETag       string
	ExpiresAt  time.Time
	mu         sync.RWMutex
}

func (c *CachedResponse) IsExpired() bool {
	if c.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(c.ExpiresAt)
}

func (c *CachedResponse) MatchesETag(ifNoneMatch string) bool {
	if ifNoneMatch == "" || c.ETag == "" {
		return false
	}
	if ifNoneMatch == "*" {
		return true
	}
	for _, etag := range splitIfNoneMatch(ifNoneMatch) {
		if etag == c.ETag || etag == "W/"+c.ETag {
			return true
		}
	}
	return false
}

func splitIfNoneMatch(header string) []string {
	result := []string{}
	current := ""
	for _, c := range header {
		if c == ',' {
			if current != "" {
				result = append(result, trim(current))
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, trim(current))
	}
	return result
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

type ResponseCacheMiddleware struct {
	mu        sync.RWMutex
	store     map[string]*CachedResponse
	maxSize   int
	defaultTTL time.Duration
}

type ResponseCacheOption func(*ResponseCacheMiddleware)

func WithResponseCacheMaxSize(size int) ResponseCacheOption {
	return func(m *ResponseCacheMiddleware) {
		m.maxSize = size
	}
}

func WithResponseCacheTTL(ttl time.Duration) ResponseCacheOption {
	return func(m *ResponseCacheMiddleware) {
		m.defaultTTL = ttl
	}
}

func NewResponseCacheMiddleware(opts ...ResponseCacheOption) *ResponseCacheMiddleware {
	m := &ResponseCacheMiddleware{
		store:      make(map[string]*CachedResponse),
		maxSize:    1000,
		defaultTTL: 5 * time.Minute,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (m *ResponseCacheMiddleware) Process(ctx *Context, next NextFunc) error {
	if ctx.Method() != "GET" && ctx.Method() != "HEAD" {
		return next(ctx)
	}
	cacheKey := ctx.Method() + ":" + ctx.Path()
	ifNoneMatch := ctx.Header("If-None-Match")
	m.mu.RLock()
	cached, ok := m.store[cacheKey]
	m.mu.RUnlock()
	if ok && !cached.IsExpired() {
		if cached.MatchesETag(ifNoneMatch) {
			ctx.response = &Response{
				StatusCode: StatusNotModified,
				Headers:    cached.Headers,
				Content:    nil,
			}
			return nil
		}
		ctx.response = &Response{
			StatusCode: cached.StatusCode,
			Headers:    cached.Headers,
			Content:    cached.Body,
		}
		return nil
	}
	err := next(ctx)
	if err != nil {
		return err
	}
	if ctx.response != nil && ctx.response.StatusCode >= 200 && ctx.response.StatusCode < 300 {
		etag := ETagCRC32(ctx.response.Content)
		m.mu.Lock()
		if len(m.store) >= m.maxSize {
			m.evictOldest()
		}
		headers := make(map[string]string)
		for k, v := range ctx.response.Headers {
			headers[k] = v
		}
		headers["ETag"] = etag
		headers["Cache-Control"] = "public, max-age=" + intToStr(int(m.defaultTTL.Seconds()))
		m.store[cacheKey] = &CachedResponse{
			StatusCode: ctx.response.StatusCode,
			Headers:    headers,
			Body:       ctx.response.Content,
			ETag:       etag,
			ExpiresAt:  time.Now().Add(m.defaultTTL),
		}
		ctx.response.Headers = headers
		m.mu.Unlock()
	}
	return nil
}

func (m *ResponseCacheMiddleware) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	for k, v := range m.store {
		if oldestKey == "" || v.ExpiresAt.Before(oldestTime) {
			oldestKey = k
			oldestTime = v.ExpiresAt
		}
	}
	if oldestKey != "" {
		delete(m.store, oldestKey)
	}
}

func (m *ResponseCacheMiddleware) Clear() {
	m.mu.Lock()
	m.store = make(map[string]*CachedResponse)
	m.mu.Unlock()
}

func (m *ResponseCacheMiddleware) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.store)
}

type ETagMiddleware struct{}

func NewETagMiddleware() *ETagMiddleware {
	return &ETagMiddleware{}
}

func (e *ETagMiddleware) Process(ctx *Context, next NextFunc) error {
	err := next(ctx)
	if err != nil {
		return err
	}
	if ctx.response == nil || len(ctx.response.Content) == 0 {
		return nil
	}
	if ctx.response.Headers["ETag"] != "" {
		return nil
	}
	etag := ETagCRC32(ctx.response.Content)
	ifNoneMatch := ctx.Header("If-None-Match")
	if ifNoneMatch != "" && (ifNoneMatch == etag || ifNoneMatch == "W/"+etag || ifNoneMatch == "*") {
		ctx.response = &Response{
			StatusCode: StatusNotModified,
			Headers:    map[string]string{"ETag": etag},
			Content:    nil,
		}
		return nil
	}
	if ctx.response.Headers == nil {
		ctx.response.Headers = make(map[string]string)
	}
	ctx.response.Headers["ETag"] = etag
	return nil
}
