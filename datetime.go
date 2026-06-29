package rivex

/*
#include "rivex_ffi.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"

// NowISO returns the current UTC time as an ISO 8601 string with microsecond
// precision (e.g. "2024-01-15T10:30:00.123456Z").
//
// The time source is the system clock via Rust's std::time::SystemTime.
func NowISO() string {
	result := C.rivex_now_iso()
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// CronNext returns the next execution time for a cron expression as an ISO 8601 string.
// Returns an empty string if the expression is invalid.
func CronNext(expr string) string {
	cExpr := C.CString(expr)
	defer C.free(unsafe.Pointer(cExpr))
	result := C.rivex_cron_next(cExpr)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// CronValidate checks whether a cron expression is valid.
func CronValidate(expr string) bool {
	cExpr := C.CString(expr)
	defer C.free(unsafe.Pointer(cExpr))
	return C.rivex_cron_validate(cExpr) == 1
}

// ConvertTimezone converts a datetime from one IANA timezone to another.
// datetimeISO is an ISO 8601 datetime string.
// fromTZ and toTZ are IANA timezone names (e.g. "America/New_York").
func ConvertTimezone(datetimeISO, fromTZ, toTZ string) string {
	cDT := C.CString(datetimeISO)
	defer C.free(unsafe.Pointer(cDT))
	cFrom := C.CString(fromTZ)
	defer C.free(unsafe.Pointer(cFrom))
	cTo := C.CString(toTZ)
	defer C.free(unsafe.Pointer(cTo))
	result := C.rivex_convert_timezone(cDT, cFrom, cTo)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}
