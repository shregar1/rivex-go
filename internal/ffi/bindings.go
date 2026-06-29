// Package ffi provides CGo bindings to the rivex-core Rust runtime.
//
// This package links against the rivex-ffi shared library (produced by the
// rivex-ffi crate in core.rivex.vexarr.com) via CGo. The C ABI exposes an opaque
// server handle, route registration, and a blocking run function that starts
// the Tokio-based Hyper HTTP server inside the Rust core.
//
// Handler callbacks are Go functions exported to C via cgo. Each registered
// Go handler is stored in a global registry keyed by an integer ID; the C
// callback trampoline looks up the handler by ID and invokes it.
package ffi

/*
#include "rivex_ffi.h"
#include <stdlib.h>
#include <string.h>

// Defined in bridge.c — wraps the Go export so we can take its address.
extern RivexResponseData* go_handler_bridge(const RivexRequestData *request, void *user_data);
*/
import "C"
import (
	"fmt"
	"sync"
	"unsafe"
)

// RequestHandler is the function signature for handling an incoming HTTP
// request from the Rust runtime.
type RequestHandler func(method, path string, headers [][2]string, body []byte, queryString string, pathParams map[string]string, clientHost string, clientPort int) RawResponse

// RawResponse is the wire format for responses passed back to the Rust
// runtime through the FFI boundary.
type RawResponse struct {
	Status  int
	Headers [][2]string
	Body    []byte
}

// handlerRegistry maps integer IDs to Go handler functions.
// The ID is passed as user_data through the C callback.
var (
	handlerMu       sync.Mutex
	handlerRegistry = make(map[uintptr]RequestHandler)
	handlerNextID   uintptr
)

func registerHandler(h RequestHandler) uintptr {
	handlerMu.Lock()
	defer handlerMu.Unlock()
	handlerNextID++
	handlerRegistry[handlerNextID] = h
	return handlerNextID
}

func lookupHandler(id uintptr) (RequestHandler, bool) {
	handlerMu.Lock()
	defer handlerMu.Unlock()
	h, ok := handlerRegistry[id]
	return h, ok
}

// goHandlerTrampoline is the C-callable function that bridges from the Rust
// runtime back into Go. It is registered as the callback for every route;
// the user_data pointer carries the handler registry ID.
//
//export goHandlerTrampoline
func goHandlerTrampoline(request *C.RivexRequestData, userData unsafe.Pointer) *C.RivexResponseData {
	id := uintptr(userData)
	handler, ok := lookupHandler(id)
	if !ok {
		body := C.CString(`{"error":"handler not found"}`)
		defer C.free(unsafe.Pointer(body))
		return C.rivex_response_json(500, body)
	}

	// Convert C request data to Go types.
	method := C.GoString(request.method)
	path := C.GoString(request.path)
	queryString := C.GoString(request.query_string)
	clientHost := C.GoString(request.client_host)
	clientPort := int(request.client_port)

	// Body
	var body []byte
	if request.body != nil && request.body_len > 0 {
		body = C.GoBytes(unsafe.Pointer(request.body), C.int(request.body_len))
	}

	// Headers
	headersLen := int(request.headers_len)
	headers := make([][2]string, 0, headersLen)
	if request.headers != nil && headersLen > 0 {
		hSlice := unsafe.Slice(request.headers, headersLen)
		for _, h := range hSlice {
			headers = append(headers, [2]string{
				C.GoString(h.name),
				C.GoString(h.value),
			})
		}
	}

	// Path params
	paramsLen := int(request.path_params_len)
	pathParams := make(map[string]string, paramsLen)
	if request.path_params != nil && paramsLen > 0 {
		pSlice := unsafe.Slice(request.path_params, paramsLen)
		for _, kv := range pSlice {
			pathParams[C.GoString(kv.key)] = C.GoString(kv.value)
		}
	}

	// Call the Go handler.
	resp := handler(method, path, headers, body, queryString, pathParams, clientHost, clientPort)

	// Build a C response. Use the Rust helper for JSON responses if the
	// content-type header is application/json; otherwise build manually.
	contentType := ""
	for _, h := range resp.Headers {
		if h[0] == "content-type" || h[0] == "Content-Type" {
			contentType = h[1]
			break
		}
	}

	var cResp *C.RivexResponseData
	if contentType == "application/json" {
		cBody := C.CString(string(resp.Body))
		cResp = C.rivex_response_json(C.uint16_t(resp.Status), cBody)
		C.free(unsafe.Pointer(cBody))
	} else if contentType == "text/plain; charset=utf-8" || contentType == "text/plain" {
		cBody := C.CString(string(resp.Body))
		cResp = C.rivex_response_text(C.uint16_t(resp.Status), cBody)
		C.free(unsafe.Pointer(cBody))
	} else {
		// Generic response: allocate body + headers manually.
		var bodyPtr *C.uint8_t
		bodyLen := len(resp.Body)
		if bodyLen > 0 {
			bodyPtr = (*C.uint8_t)(C.malloc(C.size_t(bodyLen)))
			C.memcpy(unsafe.Pointer(bodyPtr), unsafe.Pointer(&resp.Body[0]), C.size_t(bodyLen))
		}

		numHeaders := len(resp.Headers)
		var headersPtr *C.RivexHeaderPair
		if numHeaders > 0 {
			headersPtr = (*C.RivexHeaderPair)(C.malloc(C.size_t(numHeaders) * C.size_t(unsafe.Sizeof(C.RivexHeaderPair{}))))
			hSlice := unsafe.Slice(headersPtr, numHeaders)
			for i, h := range resp.Headers {
				hSlice[i].name = C.CString(h[0])
				hSlice[i].value = C.CString(h[1])
			}
		}

		cResp = (*C.RivexResponseData)(C.malloc(C.size_t(unsafe.Sizeof(C.RivexResponseData{}))))
		cResp.status_code = C.uint16_t(resp.Status)
		cResp.body = bodyPtr
		cResp.body_len = C.size_t(bodyLen)
		cResp.headers = headersPtr
		cResp.headers_len = C.size_t(numHeaders)
	}

	return cResp
}

// ServerNew creates a new Rivex server instance via the Rust FFI.
func ServerNew() unsafe.Pointer {
	return unsafe.Pointer(C.rivex_server_new())
}

// ServerFree frees a server that was never started.
func ServerFree(server unsafe.Pointer) {
	C.rivex_server_free((*C.RivexServer)(server))
}

// ServerAddRoute registers a route handler with the Rust server.
func ServerAddRoute(server unsafe.Pointer, method, path string, handler RequestHandler) {
	cMethod := C.CString(method)
	defer C.free(unsafe.Pointer(cMethod))
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	id := registerHandler(handler)

	C.rivex_server_add_route(
		(*C.RivexServer)(server),
		cMethod,
		cPath,
		C.RivexHandler(C.go_handler_bridge),
		unsafe.Pointer(id),
		200,
	)
}

// ServerRun starts the HTTP server. This call blocks until shutdown.
func ServerRun(server unsafe.Pointer, host string, port int) error {
	cHost := C.CString(host)
	defer C.free(unsafe.Pointer(cHost))

	result := C.rivex_server_run(
		(*C.RivexServer)(server),
		cHost,
		C.uint16_t(port),
	)

	if result != 0 {
		return fmt.Errorf("rivex-ffi: server exited with code %d", result)
	}
	return nil
}

// Version returns the Rivex core version string.
func Version() string {
	return C.GoString(C.rivex_version())
}
