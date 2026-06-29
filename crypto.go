package rivex

/*
#include "rivex_ffi.h"
#include <stdlib.h>
#include <string.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// SHA256 computes the SHA-256 hash of data and returns a 64-character hex string.
//
// Backed by Rust's sha2 crate for maximum performance.
func SHA256(data []byte) string {
	if len(data) == 0 {
		return sha256Empty()
	}
	result := C.rivex_sha256((*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)))
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

func sha256Empty() string {
	// SHA-256 of empty input is a well-known constant
	return "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
}

// SHA512 computes the SHA-512 hash of data and returns a 128-character hex string.
func SHA512(data []byte) string {
	if len(data) == 0 {
		return sha512Empty()
	}
	result := C.rivex_sha512((*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)))
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

func sha512Empty() string {
	return "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e"
}

// HmacSHA256 computes HMAC-SHA256 of message using key. Returns a 64-char hex string.
//
// Uses constant-time comparison internally via the Rust hmac crate.
func HmacSHA256(key, message []byte) string {
	var keyPtr, msgPtr *C.uint8_t
	if len(key) > 0 {
		keyPtr = (*C.uint8_t)(unsafe.Pointer(&key[0]))
	}
	if len(message) > 0 {
		msgPtr = (*C.uint8_t)(unsafe.Pointer(&message[0]))
	}

	result := C.rivex_hmac_sha256(
		keyPtr, C.size_t(len(key)),
		msgPtr, C.size_t(len(message)),
	)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// HmacSHA512 computes HMAC-SHA512 of message using key. Returns a 128-char hex string.
func HmacSHA512(key, message []byte) string {
	var keyPtr, msgPtr *C.uint8_t
	if len(key) > 0 {
		keyPtr = (*C.uint8_t)(unsafe.Pointer(&key[0]))
	}
	if len(message) > 0 {
		msgPtr = (*C.uint8_t)(unsafe.Pointer(&message[0]))
	}

	result := C.rivex_hmac_sha512(
		keyPtr, C.size_t(len(key)),
		msgPtr, C.size_t(len(message)),
	)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// HmacSHA256Verify verifies an HMAC-SHA256 signature using constant-time comparison.
//
// signatureHex is the expected signature as a hex string.
func HmacSHA256Verify(key, message []byte, signatureHex string) bool {
	var keyPtr, msgPtr *C.uint8_t
	if len(key) > 0 {
		keyPtr = (*C.uint8_t)(unsafe.Pointer(&key[0]))
	}
	if len(message) > 0 {
		msgPtr = (*C.uint8_t)(unsafe.Pointer(&message[0]))
	}
	cSig := C.CString(signatureHex)
	defer C.free(unsafe.Pointer(cSig))

	return C.rivex_hmac_sha256_verify(
		keyPtr, C.size_t(len(key)),
		msgPtr, C.size_t(len(message)),
		cSig,
	) == 1
}

// HmacSHA512Verify verifies an HMAC-SHA512 signature using constant-time comparison.
func HmacSHA512Verify(key, message []byte, signatureHex string) bool {
	var keyPtr, msgPtr *C.uint8_t
	if len(key) > 0 {
		keyPtr = (*C.uint8_t)(unsafe.Pointer(&key[0]))
	}
	if len(message) > 0 {
		msgPtr = (*C.uint8_t)(unsafe.Pointer(&message[0]))
	}
	cSig := C.CString(signatureHex)
	defer C.free(unsafe.Pointer(cSig))

	return C.rivex_hmac_sha512_verify(
		keyPtr, C.size_t(len(key)),
		msgPtr, C.size_t(len(message)),
		cSig,
	) == 1
}

// ---------------------------------------------------------------------------
// AES-256-GCM
// ---------------------------------------------------------------------------

// AesGcmEncrypt encrypts plaintext using AES-256-GCM.
//
// Key must be exactly 32 bytes. Returns nonce(12) || ciphertext || tag(16).
func AesGcmEncrypt(key, plaintext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("rivex: AES-256-GCM key must be exactly 32 bytes, got %d", len(key))
	}
	var ptPtr *C.uint8_t
	ptLen := C.size_t(len(plaintext))
	if len(plaintext) > 0 {
		ptPtr = (*C.uint8_t)(unsafe.Pointer(&plaintext[0]))
	}

	var outLen C.size_t
	result := C.rivex_aes_gcm_encrypt(
		(*C.uint8_t)(unsafe.Pointer(&key[0])), C.size_t(32),
		ptPtr, ptLen, &outLen,
	)
	if result == nil {
		return nil, fmt.Errorf("rivex: AES-GCM encryption failed")
	}
	defer C.rivex_bytes_free(result, outLen)
	return C.GoBytes(unsafe.Pointer(result), C.int(outLen)), nil
}

// AesGcmDecrypt decrypts AES-256-GCM ciphertext.
//
// Key must be exactly 32 bytes. Data format: nonce(12) || ciphertext || tag(16).
func AesGcmDecrypt(key, data []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("rivex: AES-256-GCM key must be exactly 32 bytes, got %d", len(key))
	}
	if len(data) < 28 {
		return nil, fmt.Errorf("rivex: ciphertext too short (need at least 28 bytes)")
	}

	var outLen C.size_t
	result := C.rivex_aes_gcm_decrypt(
		(*C.uint8_t)(unsafe.Pointer(&key[0])), C.size_t(32),
		(*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)),
		&outLen,
	)
	if result == nil {
		return nil, fmt.Errorf("rivex: AES-GCM decryption failed (invalid key or tampered data)")
	}
	defer C.rivex_bytes_free(result, outLen)
	return C.GoBytes(unsafe.Pointer(result), C.int(outLen)), nil
}

// ---------------------------------------------------------------------------
// PBKDF2
// ---------------------------------------------------------------------------

// Pbkdf2Derive derives a key from password using PBKDF2-HMAC-SHA256.
//
// Salt should be ≥16 random bytes. Iterations ≥100000 recommended.
func Pbkdf2Derive(password, salt []byte, iterations uint32, keyLength int) []byte {
	var pwPtr, sPtr *C.uint8_t
	if len(password) > 0 {
		pwPtr = (*C.uint8_t)(unsafe.Pointer(&password[0]))
	}
	if len(salt) > 0 {
		sPtr = (*C.uint8_t)(unsafe.Pointer(&salt[0]))
	}

	var outLen C.size_t
	result := C.rivex_pbkdf2_derive(
		pwPtr, C.size_t(len(password)),
		sPtr, C.size_t(len(salt)),
		C.uint32_t(iterations), C.size_t(keyLength), &outLen,
	)
	if result == nil {
		return nil
	}
	defer C.rivex_bytes_free(result, outLen)
	return C.GoBytes(unsafe.Pointer(result), C.int(outLen))
}

// Pbkdf2Verify verifies a password against a PBKDF2-derived key.
func Pbkdf2Verify(password, salt []byte, iterations uint32, expectedKey []byte) bool {
	var pwPtr, sPtr, ekPtr *C.uint8_t
	if len(password) > 0 {
		pwPtr = (*C.uint8_t)(unsafe.Pointer(&password[0]))
	}
	if len(salt) > 0 {
		sPtr = (*C.uint8_t)(unsafe.Pointer(&salt[0]))
	}
	if len(expectedKey) > 0 {
		ekPtr = (*C.uint8_t)(unsafe.Pointer(&expectedKey[0]))
	}

	return C.rivex_pbkdf2_verify(
		pwPtr, C.size_t(len(password)),
		sPtr, C.size_t(len(salt)),
		C.uint32_t(iterations),
		ekPtr, C.size_t(len(expectedKey)),
	) == 1
}

// ---------------------------------------------------------------------------
// Secure random
// ---------------------------------------------------------------------------

// RandomBytes generates n cryptographically secure random bytes.
func RandomBytes(n int) []byte {
	var outLen C.size_t
	result := C.rivex_random_bytes(C.size_t(n), &outLen)
	if result == nil {
		return nil
	}
	defer C.rivex_bytes_free(result, outLen)
	return C.GoBytes(unsafe.Pointer(result), C.int(outLen))
}

// RandomToken generates a URL-safe random token (base64url-encoded, no padding).
//
// nBytes controls entropy; default 32 → 43-char token.
func RandomToken(nBytes int) string {
	result := C.rivex_random_token(C.size_t(nBytes))
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// RandomHex generates a random hex token (2 * nBytes characters).
func RandomHex(nBytes int) string {
	result := C.rivex_random_hex(C.size_t(nBytes))
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// ---------------------------------------------------------------------------
// Ed25519
// ---------------------------------------------------------------------------

// Ed25519KeyPair holds an Ed25519 private/public key pair.
type Ed25519KeyPair struct {
	PrivateKey []byte // 32-byte seed
	PublicKey  []byte // 32-byte public key
}

// Ed25519Keygen generates a new Ed25519 keypair.
func Ed25519Keygen() *Ed25519KeyPair {
	var outLen C.size_t
	result := C.rivex_ed25519_keygen(&outLen)
	if result == nil {
		return nil
	}
	defer C.rivex_bytes_free(result, outLen)
	buf := C.GoBytes(unsafe.Pointer(result), 64)
	return &Ed25519KeyPair{
		PrivateKey: buf[:32],
		PublicKey:  buf[32:],
	}
}

// Ed25519Sign signs a message with an Ed25519 private key (32 bytes).
//
// Returns a 64-byte signature.
func Ed25519Sign(privateKey, message []byte) []byte {
	if len(privateKey) != 32 {
		return nil
	}
	var msgPtr *C.uint8_t
	if len(message) > 0 {
		msgPtr = (*C.uint8_t)(unsafe.Pointer(&message[0]))
	}

	var outLen C.size_t
	result := C.rivex_ed25519_sign(
		(*C.uint8_t)(unsafe.Pointer(&privateKey[0])),
		msgPtr, C.size_t(len(message)), &outLen,
	)
	if result == nil {
		return nil
	}
	defer C.rivex_bytes_free(result, outLen)
	return C.GoBytes(unsafe.Pointer(result), C.int(outLen))
}

// Ed25519Verify verifies an Ed25519 signature.
func Ed25519Verify(publicKey, message, signature []byte) bool {
	if len(publicKey) != 32 || len(signature) != 64 {
		return false
	}
	var msgPtr *C.uint8_t
	if len(message) > 0 {
		msgPtr = (*C.uint8_t)(unsafe.Pointer(&message[0]))
	}

	return C.rivex_ed25519_verify(
		(*C.uint8_t)(unsafe.Pointer(&publicKey[0])),
		msgPtr, C.size_t(len(message)),
		(*C.uint8_t)(unsafe.Pointer(&signature[0])), C.size_t(64),
	) == 1
}
