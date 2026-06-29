package rivex

/*
#include "rivex_ffi.h"
#include <stdlib.h>
*/
import "C"
import (
	"encoding/json"
	"runtime"
	"unsafe"
)

// ---------------------------------------------------------------------------
// Rate Limiter
// ---------------------------------------------------------------------------

// RateLimiter is a concurrent token-bucket rate limiter backed by Rust.
type RateLimiter struct {
	ptr *C.FfiRateLimiter
}

// NewRateLimiter creates a rate limiter.
// rate is tokens refilled per second, burst is the maximum bucket size.
func NewRateLimiter(rate float64, burst uint32) *RateLimiter {
	rl := &RateLimiter{ptr: C.rivex_rate_limiter_new(C.double(rate), C.uint32_t(burst))}
	runtime.SetFinalizer(rl, (*RateLimiter).Close)
	return rl
}

// Check returns true if 1 token is available for key.
func (rl *RateLimiter) Check(key string) bool {
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	return C.rivex_rate_limiter_check(rl.ptr, cKey) == 1
}

// CheckN returns true if n tokens are available for key.
func (rl *RateLimiter) CheckN(key string, n uint32) bool {
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	return C.rivex_rate_limiter_check_n(rl.ptr, cKey, C.uint32_t(n)) == 1
}

// Remaining returns the current token count for key.
func (rl *RateLimiter) Remaining(key string) float64 {
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	return float64(C.rivex_rate_limiter_remaining(rl.ptr, cKey))
}

// Reset removes the bucket for key.
func (rl *RateLimiter) Reset(key string) {
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	C.rivex_rate_limiter_reset(rl.ptr, cKey)
}

// Clear removes all buckets.
func (rl *RateLimiter) Clear() {
	C.rivex_rate_limiter_clear(rl.ptr)
}

// Close frees the rate limiter.
func (rl *RateLimiter) Close() {
	if rl.ptr != nil {
		C.rivex_rate_limiter_free(rl.ptr)
		rl.ptr = nil
	}
}

// ---------------------------------------------------------------------------
// IP / CIDR
// ---------------------------------------------------------------------------

// IPInCIDR checks if an IP address is within a CIDR range.
func IPInCIDR(ip, cidr string) bool {
	cIP := C.CString(ip)
	defer C.free(unsafe.Pointer(cIP))
	cCIDR := C.CString(cidr)
	defer C.free(unsafe.Pointer(cCIDR))
	return C.rivex_ip_in_cidr(cIP, cCIDR) == 1
}

// IsPrivateIP checks if an IP is a private/reserved address.
func IsPrivateIP(ip string) bool {
	cIP := C.CString(ip)
	defer C.free(unsafe.Pointer(cIP))
	return C.rivex_is_private_ip(cIP) == 1
}

// ---------------------------------------------------------------------------
// ETag
// ---------------------------------------------------------------------------

// ETagCRC32 generates a CRC32-based ETag for the data.
func ETagCRC32(data []byte) string {
	var ptr *C.uint8_t
	if len(data) > 0 {
		ptr = (*C.uint8_t)(unsafe.Pointer(&data[0]))
	}
	result := C.rivex_etag_crc32(ptr, C.size_t(len(data)))
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// ETagXxhash generates an xxHash64-based ETag for the data.
func ETagXxhash(data []byte) string {
	var ptr *C.uint8_t
	if len(data) > 0 {
		ptr = (*C.uint8_t)(unsafe.Pointer(&data[0]))
	}
	result := C.rivex_etag_xxhash(ptr, C.size_t(len(data)))
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// ETagWeak generates a weak CRC32-based ETag: W/"<hex>".
func ETagWeak(data []byte) string {
	var ptr *C.uint8_t
	if len(data) > 0 {
		ptr = (*C.uint8_t)(unsafe.Pointer(&data[0]))
	}
	result := C.rivex_etag_weak(ptr, C.size_t(len(data)))
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// ETagMatch compares two ETags per RFC 7232.
// If strong is true, use strong comparison; otherwise weak.
func ETagMatch(etag1, etag2 string, strong bool) bool {
	cE1 := C.CString(etag1)
	defer C.free(unsafe.Pointer(cE1))
	cE2 := C.CString(etag2)
	defer C.free(unsafe.Pointer(cE2))
	var s C.int32_t
	if strong {
		s = 1
	}
	return C.rivex_etag_match(cE1, cE2, s) == 1
}

// ParseCIDR parses a CIDR string and returns its components as JSON.
func ParseCIDR(cidr string) string {
	cCIDR := C.CString(cidr)
	defer C.free(unsafe.Pointer(cCIDR))
	result := C.rivex_parse_cidr(cCIDR)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// ---------------------------------------------------------------------------
// Body Hash / Webhook
// ---------------------------------------------------------------------------

// BodySHA256 computes SHA-256 hex digest of data.
func BodySHA256(data []byte) string {
	var ptr *C.uint8_t
	if len(data) > 0 {
		ptr = (*C.uint8_t)(unsafe.Pointer(&data[0]))
	}
	result := C.rivex_body_sha256(ptr, C.size_t(len(data)))
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// WebhookSign computes a webhook signature: sha256=<HMAC-SHA256 hex>.
func WebhookSign(secret, body []byte) string {
	var sPtr, bPtr *C.uint8_t
	if len(secret) > 0 {
		sPtr = (*C.uint8_t)(unsafe.Pointer(&secret[0]))
	}
	if len(body) > 0 {
		bPtr = (*C.uint8_t)(unsafe.Pointer(&body[0]))
	}
	result := C.rivex_webhook_sign(sPtr, C.size_t(len(secret)), bPtr, C.size_t(len(body)))
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// WebhookVerify verifies a webhook signature using constant-time comparison.
func WebhookVerify(secret, body []byte, signature string) bool {
	var sPtr, bPtr *C.uint8_t
	if len(secret) > 0 {
		sPtr = (*C.uint8_t)(unsafe.Pointer(&secret[0]))
	}
	if len(body) > 0 {
		bPtr = (*C.uint8_t)(unsafe.Pointer(&body[0]))
	}
	cSig := C.CString(signature)
	defer C.free(unsafe.Pointer(cSig))
	return C.rivex_webhook_verify(sPtr, C.size_t(len(secret)), bPtr, C.size_t(len(body)), cSig) == 1
}

// ---------------------------------------------------------------------------
// Content Negotiation
// ---------------------------------------------------------------------------

// NegotiateContentType selects the best content type from available options.
func NegotiateContentType(acceptHeader string, available []string) string {
	cHeader := C.CString(acceptHeader)
	defer C.free(unsafe.Pointer(cHeader))
	availJSON, _ := json.Marshal(available)
	cAvail := C.CString(string(availJSON))
	defer C.free(unsafe.Pointer(cAvail))
	result := C.rivex_negotiate_content_type(cHeader, cAvail)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}
