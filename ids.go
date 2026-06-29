package rivex

/*
#include "rivex_ffi.h"
#include <stdlib.h>
*/
import "C"
import (
	"encoding/json"
	"unsafe"
)

// UUIDv4 generates a random UUID v4 string.
func UUIDv4() string {
	result := C.rivex_uuid_v4()
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// UUIDv7 generates a time-ordered UUID v7 string.
func UUIDv7() string {
	result := C.rivex_uuid_v7()
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// ULID generates a ULID string (Crockford base32, lexicographically sortable).
func ULID() string {
	result := C.rivex_gen_ulid()
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// ULIDTimestamp extracts the Unix timestamp in milliseconds from a ULID string.
func ULIDTimestamp(ulidStr string) uint64 {
	cStr := C.CString(ulidStr)
	defer C.free(unsafe.Pointer(cStr))
	return uint64(C.rivex_ulid_timestamp(cStr))
}

// SnowflakeID generates a Twitter-style Snowflake ID.
// machineID identifies this worker (0-1023).
func SnowflakeID(machineID uint16) int64 {
	return int64(C.rivex_snowflake_id(C.uint16_t(machineID)))
}

// SnowflakeTimestamp extracts the Unix timestamp (ms) from a Snowflake ID.
func SnowflakeTimestamp(id int64) uint64 {
	return uint64(C.rivex_snowflake_timestamp(C.int64_t(id)))
}

// SnowflakeMachineID extracts the machine ID from a Snowflake ID.
func SnowflakeMachineID(id int64) uint16 {
	return uint16(C.rivex_snowflake_machine_id(C.int64_t(id)))
}

// NanoID generates a URL-safe unique string ID.
// Default size is 21 characters.
func NanoID(size int) string {
	result := C.rivex_nano_id(C.size_t(size))
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// NanoIDCustom generates a NanoID with a custom alphabet.
func NanoIDCustom(size int, alphabet string) string {
	cAlpha := C.CString(alphabet)
	defer C.free(unsafe.Pointer(cAlpha))
	result := C.rivex_nano_id_custom(C.size_t(size), cAlpha)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// ---------------------------------------------------------------------------
// Bulk generation (all work done in Rust, returned as JSON arrays)
// ---------------------------------------------------------------------------

func parseBulkStrings(result *C.char) []string {
	if result == nil {
		return nil
	}
	defer C.rivex_free_string(result)
	var out []string
	_ = json.Unmarshal([]byte(C.GoString(result)), &out)
	return out
}

func parseBulkInt64s(result *C.char) []int64 {
	if result == nil {
		return nil
	}
	defer C.rivex_free_string(result)
	var out []int64
	_ = json.Unmarshal([]byte(C.GoString(result)), &out)
	return out
}

// UUIDv4Bulk generates n UUID v4 strings in a single Rust call.
func UUIDv4Bulk(n int) []string {
	return parseBulkStrings(C.rivex_uuid_v4_bulk(C.size_t(n)))
}

// UUIDv7Bulk generates n time-ordered UUID v7 strings in a single Rust call.
func UUIDv7Bulk(n int) []string {
	return parseBulkStrings(C.rivex_uuid_v7_bulk(C.size_t(n)))
}

// ULIDBulk generates n ULIDs in a single Rust call.
func ULIDBulk(n int) []string {
	return parseBulkStrings(C.rivex_ulid_bulk(C.size_t(n)))
}

// SnowflakeIDBulk generates n Snowflake IDs in a single Rust call.
func SnowflakeIDBulk(n int, machineID uint16) []int64 {
	return parseBulkInt64s(C.rivex_snowflake_id_bulk(C.size_t(n), C.uint16_t(machineID)))
}

// NanoIDBulk generates n NanoIDs (default alphabet) in a single Rust call.
func NanoIDBulk(n int, size int) []string {
	return parseBulkStrings(C.rivex_nano_id_bulk(C.size_t(n), C.size_t(size)))
}
