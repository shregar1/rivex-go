package rivex

/*
#include "rivex_ffi.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"

// GzipCompress compresses data using gzip.
func GzipCompress(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	var outLen C.size_t
	ptr := C.rivex_gzip_compress(
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
		&outLen,
	)
	if ptr == nil {
		return nil
	}
	defer C.rivex_bytes_free(ptr, outLen)
	return C.GoBytes(unsafe.Pointer(ptr), C.int(outLen))
}

// GzipDecompress decompresses gzip data.
func GzipDecompress(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	var outLen C.size_t
	ptr := C.rivex_gzip_decompress(
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
		&outLen,
	)
	if ptr == nil {
		return nil
	}
	defer C.rivex_bytes_free(ptr, outLen)
	return C.GoBytes(unsafe.Pointer(ptr), C.int(outLen))
}

// BrotliCompress compresses data using Brotli.
func BrotliCompress(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	var outLen C.size_t
	ptr := C.rivex_brotli_compress(
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
		&outLen,
	)
	if ptr == nil {
		return nil
	}
	defer C.rivex_bytes_free(ptr, outLen)
	return C.GoBytes(unsafe.Pointer(ptr), C.int(outLen))
}

// BrotliDecompress decompresses Brotli data.
func BrotliDecompress(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	var outLen C.size_t
	ptr := C.rivex_brotli_decompress(
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
		&outLen,
	)
	if ptr == nil {
		return nil
	}
	defer C.rivex_bytes_free(ptr, outLen)
	return C.GoBytes(unsafe.Pointer(ptr), C.int(outLen))
}

// ZstdCompress compresses data using Zstandard.
func ZstdCompress(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	var outLen C.size_t
	ptr := C.rivex_zstd_compress(
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
		&outLen,
	)
	if ptr == nil {
		return nil
	}
	defer C.rivex_bytes_free(ptr, outLen)
	return C.GoBytes(unsafe.Pointer(ptr), C.int(outLen))
}

// ZstdDecompress decompresses Zstandard data.
func ZstdDecompress(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	var outLen C.size_t
	ptr := C.rivex_zstd_decompress(
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
		&outLen,
	)
	if ptr == nil {
		return nil
	}
	defer C.rivex_bytes_free(ptr, outLen)
	return C.GoBytes(unsafe.Pointer(ptr), C.int(outLen))
}

// LZ4Compress compresses data using LZ4.
func LZ4Compress(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	var outLen C.size_t
	ptr := C.rivex_lz4_compress(
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
		&outLen,
	)
	if ptr == nil {
		return nil
	}
	defer C.rivex_bytes_free(ptr, outLen)
	return C.GoBytes(unsafe.Pointer(ptr), C.int(outLen))
}

// LZ4Decompress decompresses LZ4 data.
func LZ4Decompress(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	var outLen C.size_t
	ptr := C.rivex_lz4_decompress(
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
		&outLen,
	)
	if ptr == nil {
		return nil
	}
	defer C.rivex_bytes_free(ptr, outLen)
	return C.GoBytes(unsafe.Pointer(ptr), C.int(outLen))
}
