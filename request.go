package rivex

import (
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

// Headers provides case-insensitive access to HTTP headers.
type Headers struct {
	items []headerEntry
}

type headerEntry struct {
	key   string
	value string
}

// NewHeaders creates a Headers from a list of key-value pairs.
func NewHeaders(pairs [][2]string) Headers {
	entries := make([]headerEntry, len(pairs))
	for i, p := range pairs {
		entries[i] = headerEntry{key: p[0], value: p[1]}
	}
	return Headers{items: entries}
}

// Get returns the first header value for the given key (case-insensitive).
// Returns empty string if the key is not found.
func (h Headers) Get(key string) string {
	target := strings.ToLower(key)
	for _, e := range h.items {
		if strings.ToLower(e.key) == target {
			return e.value
		}
	}
	return ""
}

// GetAll returns all values for the given header key (case-insensitive).
func (h Headers) GetAll(key string) []string {
	target := strings.ToLower(key)
	var result []string
	for _, e := range h.items {
		if strings.ToLower(e.key) == target {
			result = append(result, e.value)
		}
	}
	return result
}

// Has reports whether the header key exists (case-insensitive).
func (h Headers) Has(key string) bool {
	target := strings.ToLower(key)
	for _, e := range h.items {
		if strings.ToLower(e.key) == target {
			return true
		}
	}
	return false
}

// Keys returns all header keys.
func (h Headers) Keys() []string {
	keys := make([]string, len(h.items))
	for i, e := range h.items {
		keys[i] = e.key
	}
	return keys
}

// Values returns all header values.
func (h Headers) Values() []string {
	vals := make([]string, len(h.items))
	for i, e := range h.items {
		vals[i] = e.value
	}
	return vals
}

// Len returns the number of header entries.
func (h Headers) Len() int {
	return len(h.items)
}

// ToMap converts the headers to a map, keeping the last value for duplicate keys.
func (h Headers) ToMap() map[string]string {
	m := make(map[string]string, len(h.items))
	for _, e := range h.items {
		m[e.key] = e.value
	}
	return m
}

// Context is the request context passed to handlers. It provides access to the
// incoming request data and a way to store per-request state across middleware
// and handlers.
type Context struct {
	method      string
	path        string
	headers     Headers
	body        []byte
	queryString string
	pathParams  map[string]string
	clientHost  string
	clientPort  int

	store     map[string]interface{}
	storeMu   sync.RWMutex

	// response holds the response to be sent back to the client.
	// Handlers set this via JSON(), HTMLResponse(), etc.
	response  *Response

	parsedQuery     url.Values
	parsedQueryOnce sync.Once
	parsedCookies     map[string]string
	parsedCookiesOnce sync.Once
	parsedForm        url.Values
	parsedFormOnce    sync.Once
	parsedMultipart     *multipart.Form
	parsedMultipartOnce sync.Once
	parsedMultipartErr  error
}

// newContext creates a new Context from raw request data. This is an internal
// constructor used by the framework.
func newContext(method, path string, headers [][2]string, body []byte, queryString string, pathParams map[string]string, clientHost string, clientPort int) *Context {
	return &Context{
		method:      method,
		path:        path,
		headers:     NewHeaders(headers),
		body:        body,
		queryString: queryString,
		pathParams:  pathParams,
		clientHost:  clientHost,
		clientPort:  clientPort,
		store:       make(map[string]interface{}),
	}
}

// Method returns the HTTP method (GET, POST, etc.).
func (c *Context) Method() string {
	return c.method
}

// Path returns the request path without the query string.
func (c *Context) Path() string {
	return c.path
}

// Header returns the value of the named header (case-insensitive).
// Returns empty string if the header is not present.
func (c *Context) Header(name string) string {
	return c.headers.Get(name)
}

// Headers returns the full set of request headers.
func (c *Context) Headers() Headers {
	return c.headers
}

// Body returns the raw request body.
func (c *Context) Body() []byte {
	return c.body
}

// QueryString returns the raw query string without the leading '?'.
func (c *Context) QueryString() string {
	return c.queryString
}

// PathParam returns the value of a named path parameter extracted from the URL
// pattern. For example, if the route is "/users/{id}", PathParam("id") returns
// the matched segment.
func (c *Context) PathParam(name string) string {
	if c.pathParams == nil {
		return ""
	}
	return c.pathParams[name]
}

// PathParams returns all path parameters as a map.
func (c *Context) PathParams() map[string]string {
	if c.pathParams == nil {
		return map[string]string{}
	}
	cp := make(map[string]string, len(c.pathParams))
	for k, v := range c.pathParams {
		cp[k] = v
	}
	return cp
}

// QueryParam returns the first value of the named query parameter.
// Returns empty string if the parameter is not present.
func (c *Context) QueryParam(name string) string {
	c.parsedQueryOnce.Do(func() {
		c.parsedQuery, _ = url.ParseQuery(c.queryString)
	})
	return c.parsedQuery.Get(name)
}

// QueryParams returns all query parameters as a map of string slices.
func (c *Context) QueryParams() map[string][]string {
	c.parsedQueryOnce.Do(func() {
		c.parsedQuery, _ = url.ParseQuery(c.queryString)
	})
	result := make(map[string][]string, len(c.parsedQuery))
	for k, v := range c.parsedQuery {
		cp := make([]string, len(v))
		copy(cp, v)
		result[k] = cp
	}
	return result
}

// ClientHost returns the remote client's host/IP address.
func (c *Context) ClientHost() string {
	return c.clientHost
}

// ClientPort returns the remote client's port number.
func (c *Context) ClientPort() int {
	return c.clientPort
}

// URL returns the full request URL including the query string.
func (c *Context) URL() string {
	if c.queryString != "" {
		return c.path + "?" + c.queryString
	}
	return c.path
}

// ContentType returns the Content-Type header value.
func (c *Context) ContentType() string {
	return c.headers.Get("Content-Type")
}

// Cookie returns the value of the named cookie. Returns empty string if the
// cookie is not present.
func (c *Context) Cookie(name string) string {
	c.parseCookies()
	return c.parsedCookies[name]
}

// Cookies returns all cookies as a key-value map.
func (c *Context) Cookies() map[string]string {
	c.parseCookies()
	cp := make(map[string]string, len(c.parsedCookies))
	for k, v := range c.parsedCookies {
		cp[k] = v
	}
	return cp
}

func (c *Context) parseCookies() {
	c.parsedCookiesOnce.Do(func() {
		c.parsedCookies = make(map[string]string)
		cookie := c.headers.Get("Cookie")
		if cookie == "" {
			return
		}
		for _, pair := range strings.Split(cookie, ";") {
			pair = strings.TrimSpace(pair)
			if idx := strings.IndexByte(pair, '='); idx > 0 {
				key := strings.TrimSpace(pair[:idx])
				val := strings.TrimSpace(pair[idx+1:])
				c.parsedCookies[key] = val
			}
		}
	})
}

// JSON parses the request body as JSON into the given value. The value should
// be a pointer to the target type.
func (c *Context) JSON(v interface{}) error {
	return json.Unmarshal(c.body, v)
}

// Text returns the request body as a UTF-8 string.
func (c *Context) Text() string {
	return string(c.body)
}

// Bind parses the request body based on the Content-Type header and binds it
// to the given struct. Supports JSON and form-encoded bodies.
func (c *Context) Bind(v interface{}) error {
	ct := c.ContentType()
	switch {
	case strings.HasPrefix(ct, "application/json"):
		return json.Unmarshal(c.body, v)
	case strings.HasPrefix(ct, "application/x-www-form-urlencoded"):
		c.parseForm()
		return bindForm(c.parsedForm, v)
	default:
		// Default to JSON
		return json.Unmarshal(c.body, v)
	}
}

func (c *Context) parseForm() {
	c.parsedFormOnce.Do(func() {
		c.parsedForm, _ = url.ParseQuery(string(c.body))
	})
}

// FormValue returns the first value of the named form field from a
// application/x-www-form-urlencoded request body.
func (c *Context) FormValue(name string) string {
	c.parseForm()
	return c.parsedForm.Get(name)
}

// FormFile returns the uploaded file for the given form field name from a
// multipart/form-data request.
func (c *Context) FormFile(name string) (*UploadFile, error) {
	c.parseMultipart()
	if c.parsedMultipartErr != nil {
		return nil, c.parsedMultipartErr
	}
	if c.parsedMultipart == nil {
		return nil, NewHTTPException(StatusBadRequest, "no multipart data")
	}

	files := c.parsedMultipart.File[name]
	if len(files) == 0 {
		return nil, NewHTTPException(StatusBadRequest, "file not found: "+name)
	}

	fh := files[0]
	f, err := fh.Open()
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		return nil, err
	}

	return &UploadFile{
		Filename:    fh.Filename,
		ContentType: fh.Header.Get("Content-Type"),
		Headers:     headerToMap(fh.Header),
		Size:        fh.Size,
		data:        data,
	}, nil
}

func (c *Context) parseMultipart() {
	c.parsedMultipartOnce.Do(func() {
		ct := c.ContentType()
		if !strings.HasPrefix(ct, "multipart/form-data") {
			return
		}
		_, params, err := mime.ParseMediaType(ct)
		if err != nil {
			c.parsedMultipartErr = err
			return
		}
		boundary := params["boundary"]
		if boundary == "" {
			c.parsedMultipartErr = NewHTTPException(StatusBadRequest, "missing multipart boundary")
			return
		}
		reader := multipart.NewReader(strings.NewReader(string(c.body)), boundary)
		c.parsedMultipart, c.parsedMultipartErr = reader.ReadForm(32 << 20) // 32 MB
	})
}

func headerToMap(h map[string][]string) map[string]string {
	m := make(map[string]string, len(h))
	for k, v := range h {
		if len(v) > 0 {
			m[k] = v[0]
		}
	}
	return m
}

// Set stores a value in the per-request context store. This is useful for
// passing data between middleware and handlers.
func (c *Context) Set(key string, value interface{}) {
	c.storeMu.Lock()
	c.store[key] = value
	c.storeMu.Unlock()
}

// Get retrieves a value from the per-request context store.
func (c *Context) Get(key string) (interface{}, bool) {
	c.storeMu.RLock()
	v, ok := c.store[key]
	c.storeMu.RUnlock()
	return v, ok
}

// MustGet retrieves a value from the context store and panics if the key does
// not exist.
func (c *Context) MustGet(key string) interface{} {
	v, ok := c.Get(key)
	if !ok {
		panic("rivex: key " + key + " does not exist in context")
	}
	return v
}

// GetString retrieves a string value from the context store.
func (c *Context) GetString(key string) string {
	v, ok := c.Get(key)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// GetInt retrieves an integer value from the context store.
func (c *Context) GetInt(key string) int {
	v, ok := c.Get(key)
	if !ok {
		return 0
	}
	i, _ := v.(int)
	return i
}

// --- Convenience response methods on Context ---

// SendJSON serializes the given value as JSON and sets it as the response body
// with a 200 status code.
func (c *Context) SendJSON(data interface{}) error {
	c.response = JSONResponse(data)
	return nil
}

// SendJSONWithStatus serializes the given value as JSON and sets it as the
// response body with the specified status code.
func (c *Context) SendJSONWithStatus(data interface{}, statusCode int) error {
	c.response = JSONResponse(data, statusCode)
	return nil
}

// SendHTML sets an HTML response with the given content.
func (c *Context) SendHTML(content string) error {
	c.response = HTMLResponse(content)
	return nil
}

// SendText sets a plain text response with the given content.
func (c *Context) SendText(content string) error {
	c.response = TextResponse(content)
	return nil
}

// SendRedirect sends a redirect response to the given URL.
func (c *Context) SendRedirect(url string, statusCode ...int) error {
	c.response = RedirectResponse(url, statusCode...)
	return nil
}

// SetResponse sets the raw response directly.
func (c *Context) SetResponse(resp *Response) {
	c.response = resp
}

// bindForm is a simple form value to struct binder. It converts url.Values into
// JSON and then unmarshals into the target. This is a pragmatic approach that
// handles most common cases.
func bindForm(values url.Values, v interface{}) error {
	m := make(map[string]interface{}, len(values))
	for k, vals := range values {
		if len(vals) == 1 {
			// Try to parse as number or bool for better struct binding
			m[k] = tryParseValue(vals[0])
		} else {
			parsed := make([]interface{}, len(vals))
			for i, val := range vals {
				parsed[i] = tryParseValue(val)
			}
			m[k] = parsed
		}
	}
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// tryParseValue attempts to interpret a string as a number or boolean,
// falling back to the raw string.
func tryParseValue(s string) interface{} {
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}
	return s
}
