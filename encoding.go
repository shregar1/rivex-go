package rivex

/*
#include "rivex_ffi.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"

// B64Encode encodes binary data to a standard Base64 string (with padding).
func B64Encode(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	result := C.rivex_b64_encode((*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)))
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// B64Decode decodes a standard Base64 string to bytes.
// Accepts both padded and unpadded input.
//
// Returns the decoded bytes and true on success, or nil and false on error.
func B64Decode(data string) ([]byte, bool) {
	cData := C.CString(data)
	defer C.free(unsafe.Pointer(cData))

	var outLen C.size_t
	result := C.rivex_b64_decode(cData, &outLen)
	if result == nil {
		return nil, false
	}
	defer C.rivex_bytes_free(result, outLen)

	length := int(outLen)
	if length == 0 {
		return []byte{}, true
	}
	return C.GoBytes(unsafe.Pointer(result), C.int(length)), true
}

// JWTEncode creates a JWT from a JSON payload string and secret.
// expiresIn is the token lifetime in seconds (0 means no expiry).
func JWTEncode(payloadJSON, secret string, expiresIn int64) string {
	cPayload := C.CString(payloadJSON)
	defer C.free(unsafe.Pointer(cPayload))
	cSecret := C.CString(secret)
	defer C.free(unsafe.Pointer(cSecret))
	result := C.rivex_jwt_encode(cPayload, cSecret, C.int64_t(expiresIn))
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// JWTDecode decodes a JWT and returns the payload as a JSON string.
// If verify is true, the signature is validated against the secret.
func JWTDecode(token, secret string, verify bool) string {
	cToken := C.CString(token)
	defer C.free(unsafe.Pointer(cToken))
	cSecret := C.CString(secret)
	defer C.free(unsafe.Pointer(cSecret))
	v := C.int(0)
	if verify {
		v = 1
	}
	result := C.rivex_jwt_decode(cToken, cSecret, v)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// ParseCookies parses a Cookie header string into a JSON object string.
func ParseCookies(header string) string {
	cHeader := C.CString(header)
	defer C.free(unsafe.Pointer(cHeader))
	result := C.rivex_parse_cookies(cHeader)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// BuildSetCookie builds a Set-Cookie header string.
func BuildSetCookie(name, value, path string, maxAge int64, secure, httpOnly bool) string {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	cValue := C.CString(value)
	defer C.free(unsafe.Pointer(cValue))
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	s := C.int(0)
	if secure {
		s = 1
	}
	h := C.int(0)
	if httpOnly {
		h = 1
	}
	result := C.rivex_build_set_cookie(cName, cValue, cPath, C.int64_t(maxAge), s, h)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// ParseQueryString parses a URL query string into a JSON object string.
func ParseQueryString(query string) string {
	cQuery := C.CString(query)
	defer C.free(unsafe.Pointer(cQuery))
	result := C.rivex_parse_query_string(cQuery)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// BuildQueryString builds a URL query string from a JSON object string.
func BuildQueryString(json string) string {
	cJSON := C.CString(json)
	defer C.free(unsafe.Pointer(cJSON))
	result := C.rivex_build_query_string(cJSON)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}
