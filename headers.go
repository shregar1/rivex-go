package rivex

import (
	"fmt"
)

type SecurityHeadersMiddleware struct {
	config securityHeadersConfig
}

type securityHeadersConfig struct {
	xContentTypeOptions string
	xFrameOptions       string
	xssProtection       string
	hstsMaxAge          int
	hstsIncludeSubdomains bool
	hstsPreload           bool
	referrerPolicy        string
	permissionsPolicy     string
	customHeaders         map[string]string
}

type SecurityHeadersOption func(*securityHeadersConfig)

func WithXContentTypeOptions(value string) SecurityHeadersOption {
	return func(c *securityHeadersConfig) {
		c.xContentTypeOptions = value
	}
}

func WithXFrameOptions(value string) SecurityHeadersOption {
	return func(c *securityHeadersConfig) {
		c.xFrameOptions = value
	}
}

func WithXSSProtection(value string) SecurityHeadersOption {
	return func(c *securityHeadersConfig) {
		c.xssProtection = value
	}
}

func WithHSTS(maxAge int, includeSubdomains bool, preload bool) SecurityHeadersOption {
	return func(c *securityHeadersConfig) {
		c.hstsMaxAge = maxAge
		c.hstsIncludeSubdomains = includeSubdomains
		c.hstsPreload = preload
	}
}

func WithReferrerPolicy(policy string) SecurityHeadersOption {
	return func(c *securityHeadersConfig) {
		c.referrerPolicy = policy
	}
}

func WithPermissionsPolicy(policy string) SecurityHeadersOption {
	return func(c *securityHeadersConfig) {
		c.permissionsPolicy = policy
	}
}

func WithCustomSecurityHeaders(headers map[string]string) SecurityHeadersOption {
	return func(c *securityHeadersConfig) {
		if c.customHeaders == nil {
			c.customHeaders = make(map[string]string)
		}
		for k, v := range headers {
			c.customHeaders[k] = v
		}
	}
}

func defaultSecurityHeadersConfig() securityHeadersConfig {
	return securityHeadersConfig{
		xContentTypeOptions:  "nosniff",
		xFrameOptions:        "DENY",
		xssProtection:        "1; mode=block",
		hstsMaxAge:           31536000,
		hstsIncludeSubdomains: true,
		hstsPreload:           false,
		referrerPolicy:        "strict-origin-when-cross-origin",
		permissionsPolicy:     "",
	}
}

func NewSecurityHeaders(opts ...SecurityHeadersOption) *SecurityHeadersMiddleware {
	cfg := defaultSecurityHeadersConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return &SecurityHeadersMiddleware{config: cfg}
}

func (s *SecurityHeadersMiddleware) Process(ctx *Context, next NextFunc) error {
	err := next(ctx)
	if err != nil {
		return err
	}
	if ctx.response == nil {
		return nil
	}
	cfg := s.config
	if cfg.xContentTypeOptions != "" {
		ctx.response.Headers["X-Content-Type-Options"] = cfg.xContentTypeOptions
	}
	if cfg.xFrameOptions != "" {
		ctx.response.Headers["X-Frame-Options"] = cfg.xFrameOptions
	}
	if cfg.xssProtection != "" {
		ctx.response.Headers["X-XSS-Protection"] = cfg.xssProtection
	}
	if cfg.hstsMaxAge > 0 {
		hsts := fmt.Sprintf("max-age=%d", cfg.hstsMaxAge)
		if cfg.hstsIncludeSubdomains {
			hsts += "; includeSubDomains"
		}
		if cfg.hstsPreload {
			hsts += "; preload"
		}
		ctx.response.Headers["Strict-Transport-Security"] = hsts
	}
	if cfg.referrerPolicy != "" {
		ctx.response.Headers["Referrer-Policy"] = cfg.referrerPolicy
	}
	if cfg.permissionsPolicy != "" {
		ctx.response.Headers["Permissions-Policy"] = cfg.permissionsPolicy
	}
	for k, v := range cfg.customHeaders {
		ctx.response.Headers[k] = v
	}
	return nil
}

type CSPMiddleware struct {
	policy   string
	reportOnly bool
}

type CSPOption func(*CSPMiddleware)

func WithCSPPolicy(policy string) CSPOption {
	return func(c *CSPMiddleware) {
		c.policy = policy
	}
}

func WithCSPReportOnly(reportOnly bool) CSPOption {
	return func(c *CSPMiddleware) {
		c.reportOnly = reportOnly
	}
}

func NewCSP(opts ...CSPOption) *CSPMiddleware {
	c := &CSPMiddleware{
		policy: "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; form-action 'self'",
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *CSPMiddleware) Process(ctx *Context, next NextFunc) error {
	err := next(ctx)
	if err != nil {
		return err
	}
	if ctx.response != nil {
		header := "Content-Security-Policy"
		if c.reportOnly {
			header = "Content-Security-Policy-Report-Only"
		}
		ctx.response.Headers[header] = c.policy
	}
	return nil
}

func NewDefaultCSP() *CSPMiddleware {
	return NewCSP()
}

func NewRestrictiveCSP() *CSPMiddleware {
	return NewCSP(
		WithCSPPolicy("default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self'; font-src 'self'; frame-ancestors 'none'; form-action 'self'; base-uri 'none'"),
	)
}

type HSTSMiddleware struct {
	maxAge            int
	includeSubdomains bool
	preload           bool
}

type HSTSOption func(*HSTSMiddleware)

func WithHSTSMaxAge(maxAge int) HSTSOption {
	return func(h *HSTSMiddleware) {
		h.maxAge = maxAge
	}
}

func WithHSTSIncludeSubdomains(include bool) HSTSOption {
	return func(h *HSTSMiddleware) {
		h.includeSubdomains = include
	}
}

func WithHSTSPreload(preload bool) HSTSOption {
	return func(h *HSTSMiddleware) {
		h.preload = preload
	}
}

func NewHSTS(opts ...HSTSOption) *HSTSMiddleware {
	h := &HSTSMiddleware{
		maxAge:            31536000,
		includeSubdomains: true,
		preload:           false,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func (h *HSTSMiddleware) Process(ctx *Context, next NextFunc) error {
	err := next(ctx)
	if err != nil {
		return err
	}
	if ctx.response == nil {
		return nil
	}
	header := fmt.Sprintf("max-age=%d", h.maxAge)
	if h.includeSubdomains {
		header += "; includeSubDomains"
	}
	if h.preload {
		header += "; preload"
	}
	ctx.response.Headers["Strict-Transport-Security"] = header
	return nil
}

type CORP struct{}

func NewCORP() *CORP { return &CORP{} }

func (c *CORP) Process(ctx *Context, next NextFunc) error {
	err := next(ctx)
	if err != nil {
		return err
	}
	if ctx.response != nil {
		ctx.response.Headers["Cross-Origin-Resource-Policy"] = "same-origin"
	}
	return nil
}

type XFrameOptionsMiddleware struct {
	value string
}

func NewXFrameOptions(value ...string) *XFrameOptionsMiddleware {
	v := "DENY"
	if len(value) > 0 {
		v = value[0]
	}
	return &XFrameOptionsMiddleware{value: v}
}

func (x *XFrameOptionsMiddleware) Process(ctx *Context, next NextFunc) error {
	err := next(ctx)
	if err != nil {
		return err
	}
	if ctx.response != nil {
		ctx.response.Headers["X-Frame-Options"] = x.value
	}
	return nil
}

type ReferrerPolicyMiddleware struct {
	policy string
}

func NewReferrerPolicy(policy string) *ReferrerPolicyMiddleware {
	return &ReferrerPolicyMiddleware{policy: policy}
}

func (r *ReferrerPolicyMiddleware) Process(ctx *Context, next NextFunc) error {
	err := next(ctx)
	if err != nil {
		return err
	}
	if ctx.response != nil {
		ctx.response.Headers["Referrer-Policy"] = r.policy
	}
	return nil
}


