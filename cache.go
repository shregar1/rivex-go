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

// Cache is a concurrent LRU/TTL key-value cache backed by Rust's dashmap.
//
// Values are stored as strings. Use JSON serialization for complex types.
// The cache is safe for concurrent use from multiple goroutines.
type Cache struct {
	ptr *C.RivexCache
}

// NewCache creates a new cache with the given maximum size and default TTL.
//
// If maxSize is 0, the cache is unbounded. If ttlSeconds is 0, entries
// never expire unless a per-key TTL is specified.
func NewCache(maxSize int, ttlSeconds float64) *Cache {
	c := &Cache{
		ptr: C.rivex_cache_new(C.size_t(maxSize), C.double(ttlSeconds)),
	}
	runtime.SetFinalizer(c, func(c *Cache) {
		c.Close()
	})
	return c
}

// Close frees the underlying Rust cache. Safe to call multiple times.
func (c *Cache) Close() {
	if c.ptr != nil {
		C.rivex_cache_free(c.ptr)
		c.ptr = nil
	}
}

// Set stores a string value with an optional per-key TTL override.
//
// Pass ttlSeconds < 0 to use the cache default, 0 for no expiry.
func (c *Cache) Set(key, value string, ttlSeconds float64) {
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	cValue := C.CString(value)
	defer C.free(unsafe.Pointer(cValue))

	C.rivex_cache_set(c.ptr, cKey, cValue, C.double(ttlSeconds))
}

// SetJSON serializes v as JSON and stores it under key.
func (c *Cache) SetJSON(key string, v interface{}, ttlSeconds float64) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.Set(key, string(data), ttlSeconds)
	return nil
}

// Get retrieves a value by key. Returns ("", false) if missing or expired.
func (c *Cache) Get(key string) (string, bool) {
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))

	result := C.rivex_cache_get(c.ptr, cKey)
	if result == nil {
		return "", false
	}
	defer C.rivex_free_string(result)
	return C.GoString(result), true
}

// GetJSON retrieves a value and unmarshals it from JSON into dest.
func (c *Cache) GetJSON(key string, dest interface{}) bool {
	val, ok := c.Get(key)
	if !ok {
		return false
	}
	return json.Unmarshal([]byte(val), dest) == nil
}

// Delete removes a key. Returns true if the key existed.
func (c *Cache) Delete(key string) bool {
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	return C.rivex_cache_delete(c.ptr, cKey) == 1
}

// Clear removes all entries.
func (c *Cache) Clear() {
	C.rivex_cache_clear(c.ptr)
}

// Len returns the number of non-expired entries.
func (c *Cache) Len() int {
	return int(C.rivex_cache_len(c.ptr))
}

// Contains checks whether a non-expired entry exists for the given key.
func (c *Cache) Contains(key string) bool {
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	return C.rivex_cache_contains(c.ptr, cKey) == 1
}

// Keys returns all non-expired keys.
func (c *Cache) Keys() []string {
	result := C.rivex_cache_keys(c.ptr)
	defer C.rivex_free_string(result)
	jsonStr := C.GoString(result)

	var keys []string
	json.Unmarshal([]byte(jsonStr), &keys)
	return keys
}

// EvictExpired manually sweeps all expired entries.
func (c *Cache) EvictExpired() {
	C.rivex_cache_evict_expired(c.ptr)
}
