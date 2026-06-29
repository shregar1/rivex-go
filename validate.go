package rivex

/*
#include "rivex_ffi.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"

// ValidateEmail checks whether an email address has valid RFC 5322 structure.
//
// This is a syntactic check only -- no DNS or SMTP verification is performed.
// The total length must not exceed 254 octets (per RFC 5321).
func ValidateEmail(email string) bool {
	cEmail := C.CString(email)
	defer C.free(unsafe.Pointer(cEmail))

	return C.rivex_validate_email(cEmail) == 1
}

// ValidateURL checks whether a URL has a valid scheme and a non-empty host.
//
// Uses the WHATWG URL parser under the hood.
func ValidateURL(url string) bool {
	cURL := C.CString(url)
	defer C.free(unsafe.Pointer(cURL))

	return C.rivex_validate_url(cURL) == 1
}

// LuhnCheck validates a credit card number using the Luhn algorithm.
func LuhnCheck(number string) bool {
	cNum := C.CString(number)
	defer C.free(unsafe.Pointer(cNum))
	return C.rivex_luhn_check(cNum) == 1
}

// ValidateIBAN validates an International Bank Account Number.
func ValidateIBAN(iban string) bool {
	cIBAN := C.CString(iban)
	defer C.free(unsafe.Pointer(cIBAN))
	return C.rivex_validate_iban(cIBAN) == 1
}

// SanitizeHTML removes dangerous HTML tags and attributes,
// returning safe HTML suitable for rendering.
func SanitizeHTML(html string) string {
	cHTML := C.CString(html)
	defer C.free(unsafe.Pointer(cHTML))
	result := C.rivex_sanitize_html(cHTML)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}
