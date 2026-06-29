package rivex

import (
	"encoding/json"
	"io"
)

// Response represents an HTTP response to be sent back to the client.
type Response struct {
	// Content is the raw response body bytes.
	Content []byte

	// StatusCode is the HTTP status code.
	StatusCode int

	// Headers are the response headers.
	Headers map[string]string

	// MediaType is the content type of the response.
	MediaType string
}

// NewResponse creates a new Response with the given body, status code, and
// optional headers.
func NewResponse(content []byte, statusCode int, headers map[string]string) *Response {
	if headers == nil {
		headers = make(map[string]string)
	}
	return &Response{
		Content:    content,
		StatusCode: statusCode,
		Headers:    headers,
	}
}

// JSONResponse creates a JSON response by serializing the given data.
// Optionally accepts a status code (defaults to 200).
func JSONResponse(data interface{}, statusCode ...int) *Response {
	code := StatusOK
	if len(statusCode) > 0 {
		code = statusCode[0]
	}

	body, err := json.Marshal(data)
	if err != nil {
		body = []byte(`{"detail":"failed to serialize response"}`)
		code = StatusInternalServerError
	}

	return &Response{
		Content:    body,
		StatusCode: code,
		Headers:    map[string]string{"Content-Type": "application/json"},
		MediaType:  "application/json",
	}
}

// HTMLResponse creates an HTML response with the given content string.
// Optionally accepts a status code (defaults to 200).
func HTMLResponse(content string, statusCode ...int) *Response {
	code := StatusOK
	if len(statusCode) > 0 {
		code = statusCode[0]
	}

	return &Response{
		Content:    []byte(content),
		StatusCode: code,
		Headers:    map[string]string{"Content-Type": "text/html; charset=utf-8"},
		MediaType:  "text/html; charset=utf-8",
	}
}

// TextResponse creates a plain text response with the given content string.
// Optionally accepts a status code (defaults to 200).
func TextResponse(content string, statusCode ...int) *Response {
	code := StatusOK
	if len(statusCode) > 0 {
		code = statusCode[0]
	}

	return &Response{
		Content:    []byte(content),
		StatusCode: code,
		Headers:    map[string]string{"Content-Type": "text/plain; charset=utf-8"},
		MediaType:  "text/plain; charset=utf-8",
	}
}

// RedirectResponse creates a redirect response to the given URL.
// Optionally accepts a status code (defaults to 307 Temporary Redirect).
func RedirectResponse(url string, statusCode ...int) *Response {
	code := StatusTemporaryRedirect
	if len(statusCode) > 0 {
		code = statusCode[0]
	}

	return &Response{
		Content:    nil,
		StatusCode: code,
		Headers: map[string]string{
			"Location": url,
		},
	}
}

// StreamingResponse creates a response by reading all data from the given
// io.Reader. The mediaType parameter specifies the Content-Type header.
// Optionally accepts a status code (defaults to 200).
func StreamingResponse(reader io.Reader, mediaType string, statusCode ...int) *Response {
	code := StatusOK
	if len(statusCode) > 0 {
		code = statusCode[0]
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return &Response{
			Content:    []byte(`{"detail":"failed to read stream"}`),
			StatusCode: StatusInternalServerError,
			Headers:    map[string]string{"Content-Type": "application/json"},
		}
	}

	headers := make(map[string]string)
	if mediaType != "" {
		headers["Content-Type"] = mediaType
	}

	return &Response{
		Content:    data,
		StatusCode: code,
		Headers:    headers,
		MediaType:  mediaType,
	}
}
