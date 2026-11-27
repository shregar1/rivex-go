package rivex

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

const (
	contextKeyClaims    = "auth.claims"
	contextKeyUserID    = "auth.user_id"
	contextKeyRoles     = "auth.roles"
	contextKeyPrincipal = "auth.principal"
)

type Claims struct {
	Subject   string                 `json:"sub"`
	Issuer    string                 `json:"iss,omitempty"`
	Audience  string                 `json:"aud,omitempty"`
	Expiry    int64                  `json:"exp,omitempty"`
	NotBefore int64                  `json:"nbf,omitempty"`
	IssuedAt  int64                  `json:"iat,omitempty"`
	Roles     []string               `json:"roles,omitempty"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

func (c *Claims) Valid() error {
	now := time.Now().Unix()
	if c.Expiry > 0 && now >= c.Expiry {
		return NewHTTPException(StatusUnauthorized, "Token has expired")
	}
	if c.NotBefore > 0 && now < c.NotBefore {
		return NewHTTPException(StatusUnauthorized, "Token is not yet valid")
	}
	return nil
}

type JWTValidator func(claims *Claims) error

type JWTAuthMiddleware struct {
	secret      []byte
	extractor   TokenExtractor
	validators  []JWTValidator
	autoError   bool
}

type TokenExtractor func(ctx *Context) (string, error)

type JWTAuthOption func(*JWTAuthMiddleware)

func WithTokenExtractor(extractor TokenExtractor) JWTAuthOption {
	return func(j *JWTAuthMiddleware) {
		j.extractor = extractor
	}
}

func WithClaimsValidator(validators ...JWTValidator) JWTAuthOption {
	return func(j *JWTAuthMiddleware) {
		j.validators = append(j.validators, validators...)
	}
}

func WithJWTAutoError(autoError bool) JWTAuthOption {
	return func(j *JWTAuthMiddleware) {
		j.autoError = autoError
	}
}

func NewJWTAuth(secret string, opts ...JWTAuthOption) *JWTAuthMiddleware {
	j := &JWTAuthMiddleware{
		secret:    []byte(secret),
		extractor: BearerTokenExtractor,
		autoError: true,
	}
	for _, opt := range opts {
		opt(j)
	}
	return j
}

func BearerTokenExtractor(ctx *Context) (string, error) {
	auth := ctx.Header("Authorization")
	if auth == "" {
		return "", nil
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") || parts[1] == "" {
		return "", NewHTTPException(StatusUnauthorized, "Invalid authorization header").
			WithHeaders(map[string]string{"WWW-Authenticate": "Bearer"})
	}
	return parts[1], nil
}

func CookieTokenExtractor(cookieName string) TokenExtractor {
	return func(ctx *Context) (string, error) {
		token := ctx.Cookie(cookieName)
		if token == "" {
			return "", nil
		}
		return token, nil
	}
}

func QueryTokenExtractor(paramName string) TokenExtractor {
	return func(ctx *Context) (string, error) {
		token := ctx.QueryParam(paramName)
		if token == "" {
			return "", nil
		}
		return token, nil
	}
}

func (j *JWTAuthMiddleware) Process(ctx *Context, next NextFunc) error {
	token, err := j.extractor(ctx)
	if err != nil {
		return err
	}
	if token == "" {
		if j.autoError {
			return NewHTTPException(StatusUnauthorized, "Not authenticated").
				WithHeaders(map[string]string{"WWW-Authenticate": "Bearer"})
		}
		return next(ctx)
	}
	claims, err := j.decodeAndValidate(token)
	if err != nil {
		return err
	}
	for _, v := range j.validators {
		if err := v(claims); err != nil {
			return err
		}
	}
	ctx.Set(contextKeyClaims, claims)
	ctx.Set(contextKeyUserID, claims.Subject)
	if len(claims.Roles) > 0 {
		ctx.Set(contextKeyRoles, claims.Roles)
	}
	return next(ctx)
}

func (j *JWTAuthMiddleware) decodeAndValidate(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, NewHTTPException(StatusUnauthorized, "Malformed token")
	}
	sig, err := j.sign(parts[0], parts[1])
	if err != nil {
		return nil, NewHTTPException(StatusUnauthorized, "Token verification failed")
	}
	if !hmac.Equal([]byte(sig), []byte(parts[2])) {
		return nil, NewHTTPException(StatusUnauthorized, "Invalid token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, NewHTTPException(StatusUnauthorized, "Malformed token payload")
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, NewHTTPException(StatusUnauthorized, "Malformed token claims")
	}
	if err := claims.Valid(); err != nil {
		return nil, err
	}
	return &claims, nil
}

func (j *JWTAuthMiddleware) sign(header, payload string) (string, error) {
	mac := hmac.New(sha256.New, j.secret)
	if _, err := mac.Write([]byte(header + "." + payload)); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (j *JWTAuthMiddleware) GenerateToken(claims *Claims) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	sig, err := j.sign(header, payload)
	if err != nil {
		return "", err
	}
	return header + "." + payload + "." + sig, nil
}

func GetClaims(ctx *Context) *Claims {
	v, _ := ctx.Get(contextKeyClaims)
	claims, _ := v.(*Claims)
	return claims
}

func GetUserID(ctx *Context) string {
	return ctx.GetString(contextKeyUserID)
}

func GetRoles(ctx *Context) []string {
	v, ok := ctx.Get(contextKeyRoles)
	if !ok {
		return nil
	}
	roles, _ := v.([]string)
	return roles
}

func HasRole(ctx *Context, role string) bool {
	for _, r := range GetRoles(ctx) {
		if r == role {
			return true
		}
	}
	return false
}

type PermissionFunc func(ctx *Context) error

type Guard struct {
	permissions []PermissionFunc
}

func NewGuard(permissions ...PermissionFunc) *Guard {
	return &Guard{permissions: permissions}
}

func (g *Guard) Process(ctx *Context, next NextFunc) error {
	for _, p := range g.permissions {
		if err := p(ctx); err != nil {
			return err
		}
	}
	return next(ctx)
}

type PermissionOption func(*Guard)

func RequiresRoles(roles ...string) PermissionFunc {
	return func(ctx *Context) error {
		userRoles := GetRoles(ctx)
		for _, required := range roles {
			found := false
			for _, ur := range userRoles {
				if ur == required {
					found = true
					break
				}
			}
			if !found {
				return NewHTTPException(StatusForbidden, "Insufficient permissions")
			}
		}
		return nil
	}
}

func RequiresUserID() PermissionFunc {
	return func(ctx *Context) error {
		if GetUserID(ctx) == "" {
			return NewHTTPException(StatusUnauthorized, "Authentication required")
		}
		return nil
	}
}

func AllowAll() PermissionFunc {
	return func(ctx *Context) error {
		return nil
	}
}

type AuthenticatedMiddleware struct {
	autoError bool
}

func NewAuthenticated(autoError ...bool) *AuthenticatedMiddleware {
	ae := true
	if len(autoError) > 0 {
		ae = autoError[0]
	}
	return &AuthenticatedMiddleware{autoError: ae}
}

func (a *AuthenticatedMiddleware) Process(ctx *Context, next NextFunc) error {
	if GetUserID(ctx) == "" {
		if a.autoError {
			return NewHTTPException(StatusUnauthorized, "Authentication required")
		}
		return next(ctx)
	}
	return next(ctx)
}

func AdminOnly(ctx *Context, next NextFunc) error {
	roles := GetRoles(ctx)
	for _, r := range roles {
		if r == "admin" {
			return next(ctx)
		}
	}
	return NewHTTPException(StatusForbidden, "Admin access required")
}

type AuthMiddleware struct {
	extractors []TokenExtractor
	verifier   func(token string) (*Claims, error)
	autoError  bool
}

type AuthOption func(*AuthMiddleware)

func WithAuthExtractor(extractor TokenExtractor) AuthOption {
	return func(a *AuthMiddleware) {
		a.extractors = append(a.extractors, extractor)
	}
}

func WithAuthVerifier(verifier func(token string) (*Claims, error)) AuthOption {
	return func(a *AuthMiddleware) {
		a.verifier = verifier
	}
}

func WithAuthAutoError(autoError bool) AuthOption {
	return func(a *AuthMiddleware) {
		a.autoError = autoError
	}
}

func NewAuth(opts ...AuthOption) *AuthMiddleware {
	a := &AuthMiddleware{
		autoError: true,
	}
	for _, opt := range opts {
		opt(a)
	}
	if a.verifier == nil {
		a.verifier = func(token string) (*Claims, error) {
			return nil, NewHTTPException(StatusUnauthorized, "No auth verifier configured")
		}
	}
	return a
}

func (a *AuthMiddleware) Process(ctx *Context, next NextFunc) error {
	var lastErr error
	for _, extractor := range a.extractors {
		token, err := extractor(ctx)
		if err != nil {
			lastErr = err
			continue
		}
		if token == "" {
			continue
		}
		claims, err := a.verifier(token)
		if err != nil {
			lastErr = err
			continue
		}
		ctx.Set(contextKeyClaims, claims)
		ctx.Set(contextKeyUserID, claims.Subject)
		if len(claims.Roles) > 0 {
			ctx.Set(contextKeyRoles, claims.Roles)
		}
		return next(ctx)
	}
	if a.autoError && lastErr != nil {
		return lastErr
	}
	if a.autoError {
		return NewHTTPException(StatusUnauthorized, "Not authenticated")
	}
	return next(ctx)
}

type APIKeyQuery struct {
	Name       string
	AutoError  bool
	SchemeName string
	OpenAPIScheme map[string]interface{}
}

func NewAPIKeyQuery(name string, opts ...SecurityOption) *APIKeyQuery {
	cfg := defaultSecurityConfig()
	cfg.schemeName = "APIKeyQuery"
	for _, opt := range opts {
		opt(&cfg)
	}
	return &APIKeyQuery{
		Name:      name,
		AutoError: cfg.autoError,
		SchemeName: cfg.schemeName,
		OpenAPIScheme: map[string]interface{}{
			"type": "apiKey",
			"in":   "query",
			"name": name,
		},
	}
}

func (a *APIKeyQuery) Resolve(ctx *Context) (interface{}, error) {
	value := ctx.QueryParam(a.Name)
	if value == "" {
		if a.AutoError {
			return nil, NewHTTPException(StatusForbidden, "Not authenticated")
		}
		return nil, nil
	}
	return value, nil
}

type APIKeyCookie struct {
	Name       string
	AutoError  bool
	SchemeName string
	OpenAPIScheme map[string]interface{}
}

func NewAPIKeyCookie(name string, opts ...SecurityOption) *APIKeyCookie {
	cfg := defaultSecurityConfig()
	cfg.schemeName = "APIKeyCookie"
	for _, opt := range opts {
		opt(&cfg)
	}
	return &APIKeyCookie{
		Name:      name,
		AutoError: cfg.autoError,
		SchemeName: cfg.schemeName,
		OpenAPIScheme: map[string]interface{}{
			"type": "apiKey",
			"in":   "cookie",
			"name": name,
		},
	}
}

func (a *APIKeyCookie) Resolve(ctx *Context) (interface{}, error) {
	value := ctx.Cookie(a.Name)
	if value == "" {
		if a.AutoError {
			return nil, NewHTTPException(StatusForbidden, "Not authenticated")
		}
		return nil, nil
	}
	return value, nil
}

type HTTPBasic struct {
	AutoError  bool
	SchemeName string
	OpenAPIScheme map[string]interface{}
}

func NewHTTPBasic(opts ...SecurityOption) *HTTPBasic {
	cfg := defaultSecurityConfig()
	cfg.schemeName = "HTTPBasic"
	for _, opt := range opts {
		opt(&cfg)
	}
	return &HTTPBasic{
		AutoError:  cfg.autoError,
		SchemeName: cfg.schemeName,
		OpenAPIScheme: map[string]interface{}{
			"type":   "http",
			"scheme": "basic",
		},
	}
}

type BasicCredentials struct {
	Username string
	Password string
}

func (h *HTTPBasic) Resolve(ctx *Context) (interface{}, error) {
	auth := ctx.Header("Authorization")
	if auth == "" {
		if h.AutoError {
			return nil, NewHTTPException(StatusUnauthorized, "Not authenticated").
				WithHeaders(map[string]string{"WWW-Authenticate": "Basic realm=\"api\""})
		}
		return nil, nil
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "basic") {
		if h.AutoError {
			return nil, NewHTTPException(StatusUnauthorized, "Invalid authorization header")
		}
		return nil, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		if h.AutoError {
			return nil, NewHTTPException(StatusUnauthorized, "Invalid base64 encoding")
		}
		return nil, nil
	}
	pair := strings.SplitN(string(decoded), ":", 2)
	if len(pair) != 2 {
		if h.AutoError {
			return nil, NewHTTPException(StatusUnauthorized, "Invalid basic auth format")
		}
		return nil, nil
	}
	return &BasicCredentials{Username: pair[0], Password: pair[1]}, nil
}

type HTTPDigest struct {
	AutoError  bool
	SchemeName string
	OpenAPIScheme map[string]interface{}
}

func NewHTTPDigest(opts ...SecurityOption) *HTTPDigest {
	cfg := defaultSecurityConfig()
	cfg.schemeName = "HTTPDigest"
	for _, opt := range opts {
		opt(&cfg)
	}
	return &HTTPDigest{
		AutoError:  cfg.autoError,
		SchemeName: cfg.schemeName,
		OpenAPIScheme: map[string]interface{}{
			"type":   "http",
			"scheme": "digest",
		},
	}
}

func (h *HTTPDigest) Resolve(ctx *Context) (interface{}, error) {
	auth := ctx.Header("Authorization")
	if auth == "" {
		if h.AutoError {
			return nil, NewHTTPException(StatusUnauthorized, "Not authenticated")
		}
		return nil, nil
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "digest") {
		if h.AutoError {
			return nil, NewHTTPException(StatusUnauthorized, "Invalid authorization header")
		}
		return nil, nil
	}
	return parts[1], nil
}

type UserAgentMiddleware struct {
	allowed    []string
	blocked    []string
	autoError  bool
}

type UserAgentOption func(*UserAgentMiddleware)

func WithAllowedUserAgents(agents ...string) UserAgentOption {
	return func(u *UserAgentMiddleware) {
		u.allowed = agents
	}
}

func WithBlockedUserAgents(agents ...string) UserAgentOption {
	return func(u *UserAgentMiddleware) {
		u.blocked = agents
	}
}

func WithUserAgentAutoError(autoError bool) UserAgentOption {
	return func(u *UserAgentMiddleware) {
		u.autoError = autoError
	}
}

func NewUserAgentMiddleware(opts ...UserAgentOption) *UserAgentMiddleware {
	u := &UserAgentMiddleware{
		autoError: true,
	}
	for _, opt := range opts {
		opt(u)
	}
	return u
}

func (u *UserAgentMiddleware) Process(ctx *Context, next NextFunc) error {
	ua := ctx.Header("User-Agent")
	if ua == "" && u.autoError {
		return NewHTTPException(StatusBadRequest, "User-Agent header required")
	}
	for _, b := range u.blocked {
		if strings.Contains(strings.ToLower(ua), strings.ToLower(b)) {
			if u.autoError {
				return NewHTTPException(StatusForbidden, "User-Agent blocked")
			}
			return next(ctx)
		}
	}
	if len(u.allowed) > 0 {
		allowed := false
		for _, a := range u.allowed {
			if strings.Contains(strings.ToLower(ua), strings.ToLower(a)) {
				allowed = true
				break
			}
		}
		if !allowed && u.autoError {
			return NewHTTPException(StatusForbidden, "User-Agent not allowed")
		}
	}
	return next(ctx)
}

type IPFilterMiddleware struct {
	whitelist []string
	blacklist []string
	autoError bool
}

type IPFilterOption func(*IPFilterMiddleware)

func WithWhitelistedIPs(ips ...string) IPFilterOption {
	return func(i *IPFilterMiddleware) {
		i.whitelist = ips
	}
}

func WithBlacklistedIPs(ips ...string) IPFilterOption {
	return func(i *IPFilterMiddleware) {
		i.blacklist = ips
	}
}

func WithIPFilterAutoError(autoError bool) IPFilterOption {
	return func(i *IPFilterMiddleware) {
		i.autoError = autoError
	}
}

func NewIPFilterMiddleware(opts ...IPFilterOption) *IPFilterMiddleware {
	i := &IPFilterMiddleware{
		autoError: true,
	}
	for _, opt := range opts {
		opt(i)
	}
	return i
}

func (i *IPFilterMiddleware) Process(ctx *Context, next NextFunc) error {
	ip := ctx.ClientHost()
	if ip == "" {
		return next(ctx)
	}
	for _, cidr := range i.blacklist {
		if IPInCIDR(ip, cidr) {
			if i.autoError {
				return NewHTTPException(StatusForbidden, "IP blocked")
			}
			return next(ctx)
		}
	}
	if len(i.whitelist) > 0 {
		allowed := false
		for _, cidr := range i.whitelist {
			if IPInCIDR(ip, cidr) {
				allowed = true
				break
			}
		}
		if !allowed {
			if i.autoError {
				return NewHTTPException(StatusForbidden, "IP not allowed")
			}
			return next(ctx)
		}
	}
	return next(ctx)
}

type TimeoutMiddleware struct {
	timeout time.Duration
}

func NewTimeoutMiddleware(timeout time.Duration) *TimeoutMiddleware {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &TimeoutMiddleware{timeout: timeout}
}

func (t *TimeoutMiddleware) Process(ctx *Context, next NextFunc) error {
	done := make(chan error, 1)
	go func() {
		done <- next(ctx)
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(t.timeout):
		return NewHTTPException(StatusGatewayTimeout, "Request timeout")
	}
}

func Authenticated() MiddlewareFunc {
	return func(ctx *Context, next NextFunc) error {
		return NewAuthenticated().Process(ctx, next)
	}
}

func JWTRequired(secret string, opts ...JWTAuthOption) MiddlewareFunc {
	mw := NewJWTAuth(secret, opts...)
	return func(ctx *Context, next NextFunc) error {
		return mw.Process(ctx, next)
	}
}
