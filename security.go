package rivex

import (
	"strings"
)

// SecurityOption configures security scheme behavior.
type SecurityOption func(*securityConfig)

type securityConfig struct {
	autoError  bool
	schemeName string
}

func defaultSecurityConfig() securityConfig {
	return securityConfig{
		autoError: true,
	}
}

// WithAutoError controls whether the security scheme automatically returns a
// 403 error when credentials are missing. Defaults to true.
func WithAutoError(autoError bool) SecurityOption {
	return func(c *securityConfig) {
		c.autoError = autoError
	}
}

// WithSchemeName sets the OpenAPI security scheme name.
func WithSchemeName(name string) SecurityOption {
	return func(c *securityConfig) {
		c.schemeName = name
	}
}

// --- APIKeyHeader ---

// APIKeyHeader extracts an API key from a specific request header. It
// implements the Dependency interface so it can be used with Depends().
type APIKeyHeader struct {
	// Name is the header name to extract the API key from.
	Name string

	// AutoError controls whether a 403 error is automatically returned when
	// the header is missing.
	AutoError bool

	// SchemeName is the name used in OpenAPI security scheme definitions.
	SchemeName string

	// OpenAPIScheme contains the OpenAPI security scheme definition.
	OpenAPIScheme map[string]interface{}
}

// NewAPIKeyHeader creates a new API key header extractor for the given header
// name.
func NewAPIKeyHeader(name string, opts ...SecurityOption) *APIKeyHeader {
	cfg := defaultSecurityConfig()
	cfg.schemeName = "APIKeyHeader"
	for _, opt := range opts {
		opt(&cfg)
	}
	return &APIKeyHeader{
		Name:       name,
		AutoError:  cfg.autoError,
		SchemeName: cfg.schemeName,
		OpenAPIScheme: map[string]interface{}{
			"type": "apiKey",
			"in":   "header",
			"name": name,
		},
	}
}

// Resolve implements the Dependency interface. It extracts the API key from
// the configured header.
func (a *APIKeyHeader) Resolve(ctx *Context) (interface{}, error) {
	value := ctx.Header(a.Name)
	if value == "" {
		if a.AutoError {
			return nil, NewHTTPException(StatusForbidden, "Not authenticated")
		}
		return nil, nil
	}
	return value, nil
}

// --- HTTPBearer ---

// HTTPAuthorizationCredentials holds the parsed components of an HTTP
// Authorization header.
type HTTPAuthorizationCredentials struct {
	// Scheme is the authorization scheme (e.g., "Bearer").
	Scheme string

	// Credentials is the credential string (e.g., the token).
	Credentials string
}

// HTTPBearer extracts Bearer token credentials from the Authorization header.
// It implements the Dependency interface.
type HTTPBearer struct {
	// AutoError controls whether a 403 error is automatically returned when
	// credentials are missing.
	AutoError bool

	// SchemeName is the name used in OpenAPI security scheme definitions.
	SchemeName string

	// OpenAPIScheme contains the OpenAPI security scheme definition.
	OpenAPIScheme map[string]interface{}
}

// NewHTTPBearer creates a new HTTP Bearer token extractor.
func NewHTTPBearer(opts ...SecurityOption) *HTTPBearer {
	cfg := defaultSecurityConfig()
	cfg.schemeName = "HTTPBearer"
	for _, opt := range opts {
		opt(&cfg)
	}
	return &HTTPBearer{
		AutoError:  cfg.autoError,
		SchemeName: cfg.schemeName,
		OpenAPIScheme: map[string]interface{}{
			"type":   "http",
			"scheme": "bearer",
		},
	}
}

// Resolve implements the Dependency interface. It extracts and parses the
// Bearer token from the Authorization header.
func (h *HTTPBearer) Resolve(ctx *Context) (interface{}, error) {
	authorization := ctx.Header("Authorization")
	if authorization == "" {
		if h.AutoError {
			return nil, NewHTTPException(StatusForbidden, "Not authenticated")
		}
		return nil, nil
	}

	parts := strings.SplitN(authorization, " ", 2)
	if len(parts) != 2 || parts[1] == "" {
		if h.AutoError {
			return nil, NewHTTPException(StatusForbidden, "Invalid authorization header")
		}
		return nil, nil
	}

	return &HTTPAuthorizationCredentials{
		Scheme:      parts[0],
		Credentials: parts[1],
	}, nil
}

// --- OAuth2PasswordBearer ---

// OAuth2PasswordBearer extracts an OAuth2 bearer token from the Authorization
// header. It implements the Dependency interface.
type OAuth2PasswordBearer struct {
	// TokenURL is the URL of the token endpoint.
	TokenURL string

	// AutoError controls whether a 401 error is automatically returned when
	// the token is missing.
	AutoError bool

	// SchemeName is the name used in OpenAPI security scheme definitions.
	SchemeName string

	// OpenAPIScheme contains the OpenAPI security scheme definition.
	OpenAPIScheme map[string]interface{}
}

// NewOAuth2PasswordBearer creates a new OAuth2 password bearer token
// extractor.
func NewOAuth2PasswordBearer(tokenURL string, opts ...SecurityOption) *OAuth2PasswordBearer {
	cfg := defaultSecurityConfig()
	cfg.schemeName = "OAuth2PasswordBearer"
	for _, opt := range opts {
		opt(&cfg)
	}
	return &OAuth2PasswordBearer{
		TokenURL:   tokenURL,
		AutoError:  cfg.autoError,
		SchemeName: cfg.schemeName,
		OpenAPIScheme: map[string]interface{}{
			"type": "oauth2",
			"flows": map[string]interface{}{
				"password": map[string]interface{}{
					"tokenUrl": tokenURL,
					"scopes":   map[string]interface{}{},
				},
			},
		},
	}
}

// Resolve implements the Dependency interface. It extracts the OAuth2 bearer
// token from the Authorization header.
func (o *OAuth2PasswordBearer) Resolve(ctx *Context) (interface{}, error) {
	authorization := ctx.Header("Authorization")
	if authorization == "" {
		if o.AutoError {
			return nil, NewHTTPException(StatusUnauthorized, "Not authenticated").
				WithHeaders(map[string]string{"WWW-Authenticate": "Bearer"})
		}
		return nil, nil
	}

	parts := strings.SplitN(authorization, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" || parts[1] == "" {
		if o.AutoError {
			return nil, NewHTTPException(StatusUnauthorized, "Invalid authentication credentials").
				WithHeaders(map[string]string{"WWW-Authenticate": "Bearer"})
		}
		return nil, nil
	}

	return parts[1], nil
}
