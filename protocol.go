package rivex

/*
#include "rivex_ffi.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"

// ParseRawHeaders parses raw HTTP header bytes to JSON array of [name, value].
func ParseRawHeaders(data []byte) string {
	if len(data) == 0 {
		return "[]"
	}
	var ptr *C.uint8_t
	ptr = (*C.uint8_t)(unsafe.Pointer(&data[0]))
	result := C.rivex_parse_raw_headers(ptr, C.size_t(len(data)))
	if result == nil {
		return "[]"
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// ParseURL parses a URL into JSON components.
func ParseURL(urlStr string) string {
	cURL := C.CString(urlStr)
	defer C.free(unsafe.Pointer(cURL))
	result := C.rivex_parse_url(cURL)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// BuildURL builds a URL from JSON components.
func BuildURL(json string) string {
	cJSON := C.CString(json)
	defer C.free(unsafe.Pointer(cJSON))
	result := C.rivex_build_url(cJSON)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// ResolveURL resolves a relative URL against a base URL.
func ResolveURL(base, relative string) string {
	cBase := C.CString(base)
	defer C.free(unsafe.Pointer(cBase))
	cRel := C.CString(relative)
	defer C.free(unsafe.Pointer(cRel))
	result := C.rivex_resolve_url(cBase, cRel)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// SniffMimeType detects MIME type from file magic bytes.
func SniffMimeType(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	ptr := (*C.uint8_t)(unsafe.Pointer(&data[0]))
	result := C.rivex_sniff_mime_type(ptr, C.size_t(len(data)))
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// MimeFromExtension looks up MIME type from file extension.
func MimeFromExtension(ext string) string {
	cExt := C.CString(ext)
	defer C.free(unsafe.Pointer(cExt))
	result := C.rivex_mime_from_extension(cExt)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// ExtensionFromMime looks up file extension from MIME type.
func ExtensionFromMime(mime string) string {
	cMime := C.CString(mime)
	defer C.free(unsafe.Pointer(cMime))
	result := C.rivex_extension_from_mime(cMime)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// ParseGraphQL parses a GraphQL query into a JSON AST.
func ParseGraphQL(query string) string {
	cQuery := C.CString(query)
	defer C.free(unsafe.Pointer(cQuery))
	result := C.rivex_parse_graphql(cQuery)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// ValidateGraphQL validates a GraphQL query syntax.
func ValidateGraphQL(query string) bool {
	cQuery := C.CString(query)
	defer C.free(unsafe.Pointer(cQuery))
	return C.rivex_validate_graphql(cQuery) == 1
}
