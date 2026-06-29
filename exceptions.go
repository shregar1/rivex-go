package rivex

import (
	"encoding/json"
	"fmt"
)

// HTTPException represents an HTTP error that should be returned to the client.
// It implements the error interface and can be returned from handlers to trigger
// the framework's error handling.
type HTTPException struct {
	// StatusCode is the HTTP status code for this error.
	StatusCode int

	// Detail is the error detail, which can be a string, map, or any
	// JSON-serializable value.
	Detail interface{}

	// Headers are optional response headers to include with the error response.
	Headers map[string]string
}

// NewHTTPException creates a new HTTPException with the given status code and
// optional detail message. If no detail is provided, a default phrase for the
// status code is used.
func NewHTTPException(statusCode int, detail ...interface{}) *HTTPException {
	var d interface{}
	if len(detail) > 0 && detail[0] != nil {
		d = detail[0]
	} else {
		d = defaultPhrase(statusCode)
	}
	return &HTTPException{
		StatusCode: statusCode,
		Detail:     d,
	}
}

// WithHeaders returns a copy of the exception with the given headers set.
func (e *HTTPException) WithHeaders(headers map[string]string) *HTTPException {
	e.Headers = headers
	return e
}

// Error implements the error interface.
func (e *HTTPException) Error() string {
	return fmt.Sprintf("HTTP %d: %v", e.StatusCode, e.Detail)
}

// toResponse converts the HTTPException into a Response suitable for sending
// to the client.
func (e *HTTPException) toResponse() *Response {
	body := map[string]interface{}{"detail": e.Detail}
	data, err := json.Marshal(body)
	if err != nil {
		data = []byte(`{"detail":"Internal Server Error"}`)
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}
	for k, v := range e.Headers {
		headers[k] = v
	}

	return NewResponse(data, e.StatusCode, headers)
}

// defaultPhrase returns the standard reason phrase for common HTTP error codes.
func defaultPhrase(code int) string {
	phrase := StatusText(code)
	if phrase != "" {
		return phrase
	}
	return "Error"
}
