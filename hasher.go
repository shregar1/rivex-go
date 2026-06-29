package rivex

/*
#include "rivex_ffi.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"

// HashPassword hashes a password using the specified algorithm.
// Supported algorithms are "argon2" and "bcrypt".
//
// Returns the hashed password string (e.g. "$argon2id$v=19$..." or "$2b$12$..."),
// or an empty string if hashing fails.
func HashPassword(password, algorithm string) string {
	cPassword := C.CString(password)
	defer C.free(unsafe.Pointer(cPassword))
	cAlgorithm := C.CString(algorithm)
	defer C.free(unsafe.Pointer(cAlgorithm))

	result := C.rivex_hash_password(cPassword, cAlgorithm)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// VerifyPassword verifies a plaintext password against a hash string.
// The algorithm is auto-detected from the hash prefix ($argon2 for Argon2id,
// $2b$/$2a$/$2y$ for bcrypt).
//
// Returns true if the password matches the hash.
func VerifyPassword(password, hash string) bool {
	cPassword := C.CString(password)
	defer C.free(unsafe.Pointer(cPassword))
	cHash := C.CString(hash)
	defer C.free(unsafe.Pointer(cHash))

	return C.rivex_verify_password(cPassword, cHash) == 1
}
