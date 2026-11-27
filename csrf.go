package rivex

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

const (
	csrfTokenLength = 32
	csrfContextKey  = "csrf.token"
)

type CSRFMiddleware struct {
	cookieName  string
	headerName  string
	secret      []byte
	safeMethods map[string]bool
	autoError   bool
}

type CSRFOption func(*CSRFMiddleware)

func WithCSRFCookieName(name string) CSRFOption {
	return func(c *CSRFMiddleware) {
		c.cookieName = name
	}
}

func WithCSRFHeaderName(name string) CSRFOption {
	return func(c *CSRFMiddleware) {
		c.headerName = name
	}
}

func WithCSRFSecret(secret string) CSRFOption {
	return func(c *CSRFMiddleware) {
		if secret != "" {
			h := sha256.Sum256([]byte(secret))
			c.secret = h[:]
		}
	}
}

func WithCSRFExemptMethods(methods ...string) CSRFOption {
	return func(c *CSRFMiddleware) {
		for _, m := range methods {
			c.safeMethods[strings.ToUpper(m)] = true
		}
	}
}

func WithCSRFAutoError(autoError bool) CSRFOption {
	return func(c *CSRFMiddleware) {
		c.autoError = autoError
	}
}

func NewCSRF(opts ...CSRFOption) *CSRFMiddleware {
	c := &CSRFMiddleware{
		cookieName:  "csrf_token",
		headerName:  "X-CSRF-Token",
		safeMethods: map[string]bool{"GET": true, "HEAD": true, "OPTIONS": true, "TRACE": true},
		autoError:   true,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.secret == nil {
		key := make([]byte, 32)
		rand.Read(key)
		c.secret = key
	}
	return c
}

func (c *CSRFMiddleware) Process(ctx *Context, next NextFunc) error {
	if c.safeMethods[strings.ToUpper(ctx.Method())] {
		csrfToken, err := c.generateToken(ctx)
		if err != nil {
			return err
		}
		ctx.Set(csrfContextKey, csrfToken)
		err = next(ctx)
		if err != nil {
			return err
		}
		if ctx.response != nil {
			ctx.response.Headers["Set-Cookie"] = c.cookieName + "=" + csrfToken + "; Path=/; HttpOnly; SameSite=Lax"
			ctx.response.Headers["X-CSRF-Token"] = csrfToken
		}
		return nil
	}
	cookieToken := ctx.Cookie(c.cookieName)
	headerToken := ctx.Header(c.headerName)
	if cookieToken == "" || headerToken == "" {
		if c.autoError {
			return NewHTTPException(StatusForbidden, "CSRF token missing")
		}
		return next(ctx)
	}
	if !c.verifyTokens(cookieToken, headerToken) {
		if c.autoError {
			return NewHTTPException(StatusForbidden, "CSRF token mismatch")
		}
		return next(ctx)
	}
	ctx.Set(csrfContextKey, headerToken)
	return next(ctx)
}

func (c *CSRFMiddleware) generateToken(ctx *Context) (string, error) {
	raw := make([]byte, csrfTokenLength)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	mac, err := c.sign(raw)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(mac), nil
}

func (c *CSRFMiddleware) sign(data []byte) ([]byte, error) {
	mac := sha256.Sum256(append(c.secret, data...))
	return append(data, mac[:]...), nil
}

func (c *CSRFMiddleware) verifyTokens(cookieToken, headerToken string) bool {
	cookieRaw, err := base64.RawURLEncoding.DecodeString(cookieToken)
	if err != nil {
		return false
	}
	headerRaw, err := base64.RawURLEncoding.DecodeString(headerToken)
	if err != nil {
		return false
	}
	if len(cookieRaw) < csrfTokenLength+sha256.Size || len(headerRaw) < csrfTokenLength+sha256.Size {
		return false
	}
	cookiePayload := cookieRaw[:csrfTokenLength]
	headerPayload := headerRaw[:csrfTokenLength]
	if !hmacEqual(cookiePayload, headerPayload) {
		return false
	}
	expectedMAC := sha256.Sum256(append(c.secret, cookiePayload...))
	actualMAC := cookieRaw[csrfTokenLength:]
	return hmacEqual(expectedMAC[:], actualMAC)
}

func hmacEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	result := byte(0)
	for i := range a {
		result |= a[i] ^ b[i]
	}
	return result == 0
}

func GetCSRFToken(ctx *Context) string {
	v, ok := ctx.Get(csrfContextKey)
	if !ok {
		return ""
	}
	token, _ := v.(string)
	return token
}

type CSRFExemptMiddleware struct {
	paths map[string]bool
}

func NewCSRFExempt(paths ...string) *CSRFExemptMiddleware {
	p := make(map[string]bool)
	for _, path := range paths {
		p[path] = true
	}
	return &CSRFExemptMiddleware{paths: p}
}

func (c *CSRFExemptMiddleware) Process(ctx *Context, next NextFunc) error {
	if c.paths[ctx.Path()] {
		ctx.Set(csrfContextKey, "exempt")
	}
	return next(ctx)
}

func GenerateCSRFToken() string {
	raw := make([]byte, csrfTokenLength)
	rand.Read(raw)
	return hex.EncodeToString(raw)
}
