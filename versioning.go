package rivex

import (
	"strconv"
	"strings"
)

type Version struct {
	Major int
	Minor int
	Patch int
	Label string
}

type VersioningScheme string

const (
	VersionHeader  VersioningScheme = "header"
	VersionQuery   VersioningScheme = "query"
	VersionPath    VersioningScheme = "path"
	VersionCookie  VersioningScheme = "cookie"
	VersionMedia   VersioningScheme = "media_type"
)

type VersioningConfig struct {
	Scheme         VersioningScheme
	ParamName      string
	DefaultVersion string
	Strict         bool
}

type VersioningOption func(*VersioningConfig)

func WithVersionScheme(scheme VersioningScheme) VersioningOption {
	return func(v *VersioningConfig) {
		v.Scheme = scheme
	}
}

func WithVersionParam(name string) VersioningOption {
	return func(v *VersioningConfig) {
		v.ParamName = name
	}
}

func WithDefaultVersion(version string) VersioningOption {
	return func(v *VersioningConfig) {
		v.DefaultVersion = version
	}
}

func WithVersionStrict(strict bool) VersioningOption {
	return func(v *VersioningConfig) {
		v.Strict = strict
	}
}

func NewVersioningConfig(opts ...VersioningOption) *VersioningConfig {
	cfg := &VersioningConfig{
		Scheme:         VersionHeader,
		ParamName:      "X-API-Version",
		DefaultVersion: "1.0.0",
		Strict:         true,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

type VersionMiddleware struct {
	config *VersioningConfig
}

func NewVersionMiddleware(opts ...VersioningOption) *VersionMiddleware {
	return &VersionMiddleware{
		config: NewVersioningConfig(opts...),
	}
}

func (vm *VersionMiddleware) Process(ctx *Context, next NextFunc) error {
	version := vm.extractVersion(ctx)
	if version == "" {
		if vm.config.Strict {
			return NewHTTPException(StatusBadRequest, "API version is required")
		}
		version = vm.config.DefaultVersion
	}
	parsed, err := ParseVersion(version)
	if err != nil {
		if vm.config.Strict {
			return NewHTTPException(StatusBadRequest, "Invalid API version: "+version)
		}
		parsed = &Version{Major: 1}
	}
	ctx.Set("api.version", version)
	ctx.Set("api.version.parsed", parsed)
	return next(ctx)
}

func (vm *VersionMiddleware) extractVersion(ctx *Context) string {
	switch vm.config.Scheme {
	case VersionHeader:
		v := ctx.Header(vm.config.ParamName)
		if v == "" {
			v = ctx.Header("Accept-Version")
		}
		return v
	case VersionQuery:
		return ctx.QueryParam(vm.config.ParamName)
	case VersionCookie:
		return ctx.Cookie(vm.config.ParamName)
	case VersionMedia:
		accept := ctx.Header("Accept")
		return extractVersionFromMediaType(accept, vm.config.ParamName)
	case VersionPath:
		return ctx.PathParam(vm.config.ParamName)
	default:
		return ctx.Header(vm.config.ParamName)
	}
}

func extractVersionFromMediaType(accept, paramName string) string {
	parts := strings.Split(accept, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "v=") || strings.HasPrefix(part, "version=") {
			if idx := strings.IndexByte(part, '='); idx > 0 {
				return strings.TrimSpace(part[idx+1:])
			}
		}
	}
	return ""
}

func ParseVersion(s string) (*Version, error) {
	s = strings.TrimSpace(s)
	label := ""
	if idx := strings.IndexAny(s, "-+"); idx > 0 {
		label = s[idx:]
		s = s[:idx]
	}
	parts := strings.Split(s, ".")
	v := &Version{Label: label}
	for i, part := range parts {
		num, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		switch i {
		case 0:
			v.Major = num
		case 1:
			v.Minor = num
		case 2:
			v.Patch = num
		}
	}
	return v, nil
}

func (v *Version) String() string {
	s := strconv.Itoa(v.Major)
	if v.Minor > 0 || v.Patch > 0 {
		s += "." + strconv.Itoa(v.Minor)
		if v.Patch > 0 {
			s += "." + strconv.Itoa(v.Patch)
		}
	}
	if v.Label != "" {
		s += v.Label
	}
	return s
}

func (v *Version) Compare(other *Version) int {
	if v.Major != other.Major {
		return v.Major - other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor - other.Minor
	}
	if v.Patch != other.Patch {
		return v.Patch - other.Patch
	}
	return 0
}

func (v *Version) Satisfies(constraint string) bool {
	constraint = strings.TrimSpace(constraint)
	if constraint == "" || constraint == "*" {
		return true
	}
	if strings.HasPrefix(constraint, ">=") {
		minVer, err := ParseVersion(constraint[2:])
		if err != nil {
			return false
		}
		return v.Compare(minVer) >= 0
	}
	if strings.HasPrefix(constraint, "^") {
		minVer, err := ParseVersion(constraint[1:])
		if err != nil {
			return false
		}
		return v.Major == minVer.Major && v.Compare(minVer) >= 0
	}
	if strings.HasPrefix(constraint, "~") {
		minVer, err := ParseVersion(constraint[1:])
		if err != nil {
			return false
		}
		return v.Major == minVer.Major && v.Minor == minVer.Minor && v.Compare(minVer) >= 0
	}
	exact, err := ParseVersion(constraint)
	if err != nil {
		return false
	}
	return v.Compare(exact) == 0
}

func GetVersion(ctx *Context) string {
	return ctx.GetString("api.version")
}

func GetParsedVersion(ctx *Context) *Version {
	v, ok := ctx.Get("api.version.parsed")
	if !ok {
		return &Version{Major: 1}
	}
	ver, _ := v.(*Version)
	return ver
}

type VersionedRoute struct {
	Route    RouteRecord
	Version  string
}

type VersionedRouter struct {
	routes  map[string][]VersionedRoute
	config  *VersioningConfig
}

func NewVersionedRouter(opts ...VersioningOption) *VersionedRouter {
	return &VersionedRouter{
		routes: make(map[string][]VersionedRoute),
		config: NewVersioningConfig(opts...),
	}
}

func (vr *VersionedRouter) Add(method, path string, version string, handler HandlerFunc, opts ...RouteOption) {
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
	vr.routes[version] = append(vr.routes[version], VersionedRoute{Route: record, Version: version})
}

func (vr *VersionedRouter) Routes() []RouteRecord {
	var all []RouteRecord
	for _, versionRoutes := range vr.routes {
		for _, vr := range versionRoutes {
			all = append(all, vr.Route)
		}
	}
	return all
}

func (vr *VersionedRouter) RouteForVersion(version string) []RouteRecord {
	routes, ok := vr.routes[version]
	if !ok {
		return nil
	}
	result := make([]RouteRecord, len(routes))
	for i, r := range routes {
		result[i] = r.Route
	}
	return result
}

type VersionNegotiator struct {
	supported map[string]bool
	defaultVer string
}

func NewVersionNegotiator(supported []string, defaultVer string) *VersionNegotiator {
	m := make(map[string]bool)
	for _, v := range supported {
		m[v] = true
	}
	return &VersionNegotiator{
		supported:  m,
		defaultVer: defaultVer,
	}
}

func (n *VersionNegotiator) Negotiate(ctx *Context) string {
	version := GetVersion(ctx)
	if version == "" {
		return n.defaultVer
	}
	if n.supported[version] {
		return version
	}
	if n.supported["v"+version] {
		return "v" + version
	}
	return n.defaultVer
}

func DeprecateVersion(version string, sunsetDate string, helpURL string) MiddlewareFunc {
	return func(ctx *Context, next NextFunc) error {
		v := GetVersion(ctx)
		if v == version {
			if sunsetDate != "" {
				ctx.response.Headers["Sunset"] = sunsetDate
			}
			if helpURL != "" {
				ctx.response.Headers["Deprecation"] = "true"
			}
		}
		return next(ctx)
	}
}
