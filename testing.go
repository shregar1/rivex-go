package rivex

import (
	"encoding/json"
	"net/url"
	"strings"
)

// TestClient provides an in-process HTTP client for testing Rivex applications.
// It dispatches requests directly to the application's handler chain without
// starting a real HTTP server, making tests fast and deterministic.
type TestClient struct {
	app *Rivex
}

// NewTestClient creates a new TestClient for the given application.
func NewTestClient(app *Rivex) *TestClient {
	return &TestClient{app: app}
}

// RequestOption configures a test request.
type RequestOption func(*testRequest)

type testRequest struct {
	headers     map[string]string
	params      map[string]string
	jsonBody    interface{}
	formData    map[string]string
	body        []byte
	contentType string
}

// WithHeader adds a header to the test request.
func WithHeader(key, value string) RequestOption {
	return func(r *testRequest) {
		r.headers[key] = value
	}
}

// WithHeaders sets multiple headers on the test request.
func WithHeaders(headers map[string]string) RequestOption {
	return func(r *testRequest) {
		for k, v := range headers {
			r.headers[k] = v
		}
	}
}

// WithQueryParams adds query parameters to the test request.
func WithQueryParams(params map[string]string) RequestOption {
	return func(r *testRequest) {
		r.params = params
	}
}

// WithJSONBody sets the request body as JSON.
func WithJSONBody(body interface{}) RequestOption {
	return func(r *testRequest) {
		r.jsonBody = body
	}
}

// WithFormData sets the request body as form-encoded data.
func WithFormData(data map[string]string) RequestOption {
	return func(r *testRequest) {
		r.formData = data
	}
}

// WithBody sets the raw request body.
func WithBody(body []byte) RequestOption {
	return func(r *testRequest) {
		r.body = body
	}
}

// WithContentType sets the Content-Type header.
func WithContentType(ct string) RequestOption {
	return func(r *testRequest) {
		r.contentType = ct
	}
}

// TestResponse holds the result of a test request.
type TestResponse struct {
	// StatusCode is the HTTP status code of the response.
	StatusCode int

	// Headers contains the response headers.
	Headers map[string]string

	// Body is the raw response body.
	Body []byte
}

// JSON unmarshals the response body into the given value.
func (tr *TestResponse) JSON(v interface{}) error {
	return json.Unmarshal(tr.Body, v)
}

// Text returns the response body as a string.
func (tr *TestResponse) Text() string {
	return string(tr.Body)
}

// JSONMap parses the response body as a JSON object and returns it as a
// map[string]interface{}. Returns an error if the body is not valid JSON.
func (tr *TestResponse) JSONMap() (map[string]interface{}, error) {
	var m map[string]interface{}
	err := json.Unmarshal(tr.Body, &m)
	return m, err
}

// Get sends a GET request to the given URL path.
func (tc *TestClient) Get(urlPath string, opts ...RequestOption) *TestResponse {
	return tc.Request("GET", urlPath, opts...)
}

// Post sends a POST request to the given URL path.
func (tc *TestClient) Post(urlPath string, opts ...RequestOption) *TestResponse {
	return tc.Request("POST", urlPath, opts...)
}

// Put sends a PUT request to the given URL path.
func (tc *TestClient) Put(urlPath string, opts ...RequestOption) *TestResponse {
	return tc.Request("PUT", urlPath, opts...)
}

// Delete sends a DELETE request to the given URL path.
func (tc *TestClient) Delete(urlPath string, opts ...RequestOption) *TestResponse {
	return tc.Request("DELETE", urlPath, opts...)
}

// Patch sends a PATCH request to the given URL path.
func (tc *TestClient) Patch(urlPath string, opts ...RequestOption) *TestResponse {
	return tc.Request("PATCH", urlPath, opts...)
}

// Request sends an HTTP request with the given method and URL path.
func (tc *TestClient) Request(method, urlPath string, opts ...RequestOption) *TestResponse {
	req := &testRequest{
		headers: make(map[string]string),
	}
	for _, opt := range opts {
		opt(req)
	}

	// Parse the URL
	parsedURL, err := url.Parse(urlPath)
	if err != nil {
		return &TestResponse{
			StatusCode: StatusBadRequest,
			Headers:    map[string]string{},
			Body:       []byte(`{"detail":"invalid URL"}`),
		}
	}

	path := parsedURL.Path
	if path == "" {
		path = "/"
	}
	queryString := parsedURL.RawQuery

	// Append query params
	if req.params != nil {
		vals := url.Values{}
		for k, v := range req.params {
			vals.Set(k, v)
		}
		extra := vals.Encode()
		if queryString != "" {
			queryString += "&" + extra
		} else {
			queryString = extra
		}
	}

	// Build headers
	var headerPairs [][2]string
	for k, v := range req.headers {
		headerPairs = append(headerPairs, [2]string{strings.ToLower(k), v})
	}

	// Build body
	var body []byte
	if req.jsonBody != nil {
		body, _ = json.Marshal(req.jsonBody)
		headerPairs = append(headerPairs, [2]string{"content-type", "application/json"})
	} else if req.formData != nil {
		vals := url.Values{}
		for k, v := range req.formData {
			vals.Set(k, v)
		}
		body = []byte(vals.Encode())
		headerPairs = append(headerPairs, [2]string{"content-type", "application/x-www-form-urlencoded"})
	} else if req.body != nil {
		body = req.body
		if req.contentType != "" {
			headerPairs = append(headerPairs, [2]string{"content-type", req.contentType})
		}
	}

	// Find matching route
	handler, pathParams := tc.resolveHandler(method, path)

	// Build context
	ctx := newContext(method, path, headerPairs, body, queryString, pathParams, "127.0.0.1", 0)

	// Execute the handler chain
	execErr := handler(ctx)
	if execErr != nil {
		return tc.errorToResponse(execErr)
	}

	resp := ctx.response
	if resp == nil {
		resp = NewResponse(nil, StatusNoContent, nil)
	}

	return &TestResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Headers,
		Body:       resp.Content,
	}
}

// resolveHandler finds the matching route handler and extracts path parameters.
func (tc *TestClient) resolveHandler(method, path string) (HandlerFunc, map[string]string) {
	for _, route := range tc.app.routes {
		if route.Method != method {
			continue
		}

		params, ok := matchPath(route.Path, path)
		if !ok {
			continue
		}

		// Build the full handler chain with middleware and error handling
		final := func(ctx *Context) error {
			err := route.Handler(ctx)
			if err != nil {
				return err
			}
			if ctx.response == nil {
				ctx.response = NewResponse(nil, StatusNoContent, nil)
			}
			// Apply route-level status code override
			if route.StatusCode != StatusOK && ctx.response.StatusCode == StatusOK {
				ctx.response.StatusCode = route.StatusCode
			}
			return nil
		}

		handler := tc.app.applyMiddleware(final)
		return handler, params
	}

	// No matching route found
	notFound := func(ctx *Context) error {
		body, _ := json.Marshal(map[string]string{"detail": "Not Found"})
		ctx.response = NewResponse(body, StatusNotFound, map[string]string{
			"Content-Type": "application/json",
		})
		return nil
	}
	return notFound, nil
}

// errorToResponse converts an error into a TestResponse.
func (tc *TestClient) errorToResponse(err error) *TestResponse {
	if httpErr, ok := err.(*HTTPException); ok {
		resp := httpErr.toResponse()
		return &TestResponse{
			StatusCode: resp.StatusCode,
			Headers:    resp.Headers,
			Body:       resp.Content,
		}
	}

	detail := "Internal Server Error"
	if tc.app.debug {
		detail = err.Error()
	}
	body, _ := json.Marshal(map[string]interface{}{"detail": detail})
	return &TestResponse{
		StatusCode: StatusInternalServerError,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       body,
	}
}
