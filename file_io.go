package rivex

/*
#include "rivex_ffi.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"

// ChecksumMD5 computes the MD5 hex digest of the given data.
func ChecksumMD5(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	result := C.rivex_checksum_md5(
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
	)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// ChecksumSHA256 computes the SHA-256 hex digest of the given data.
func ChecksumSHA256(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	result := C.rivex_checksum_sha256(
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
	)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// ChecksumCRC32 computes the CRC32 hex digest of the given data.
func ChecksumCRC32(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	result := C.rivex_checksum_crc32(
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
	)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// ZipCreate creates a zip archive from a JSON array of entries.
// Each entry is {"path": "...", "data_b64": "..."}.
// Returns the raw zip bytes.
func ZipCreate(entriesJSON string) []byte {
	cJSON := C.CString(entriesJSON)
	defer C.free(unsafe.Pointer(cJSON))
	var outLen C.size_t
	ptr := C.rivex_zip_create(cJSON, &outLen)
	if ptr == nil {
		return nil
	}
	defer C.rivex_bytes_free(ptr, outLen)
	return C.GoBytes(unsafe.Pointer(ptr), C.int(outLen))
}

// ZipExtract extracts a zip archive and returns a JSON array of entries.
// Each entry is {"path": "...", "data_b64": "..."}.
func ZipExtract(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	result := C.rivex_zip_extract(
		(*C.uint8_t)(unsafe.Pointer(&data[0])),
		C.size_t(len(data)),
	)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// ImageResize resizes an image. Input/output are PNG bytes.
func ImageResize(data []byte, width, height uint32) []byte {
	if len(data) == 0 {
		return nil
	}
	ptr := (*C.uint8_t)(unsafe.Pointer(&data[0]))
	var outLen C.size_t
	result := C.rivex_image_resize(ptr, C.size_t(len(data)), C.uint32_t(width), C.uint32_t(height), &outLen)
	if result == nil {
		return nil
	}
	defer C.rivex_bytes_free(result, outLen)
	return C.GoBytes(unsafe.Pointer(result), C.int(outLen))
}

// ImageThumbnail creates a thumbnail preserving aspect ratio.
func ImageThumbnail(data []byte, maxDim uint32) []byte {
	if len(data) == 0 {
		return nil
	}
	ptr := (*C.uint8_t)(unsafe.Pointer(&data[0]))
	var outLen C.size_t
	result := C.rivex_image_thumbnail(ptr, C.size_t(len(data)), C.uint32_t(maxDim), &outLen)
	if result == nil {
		return nil
	}
	defer C.rivex_bytes_free(result, outLen)
	return C.GoBytes(unsafe.Pointer(result), C.int(outLen))
}

// ImageDimensions returns image width and height.
func ImageDimensions(data []byte) (width, height uint32, ok bool) {
	if len(data) == 0 {
		return 0, 0, false
	}
	ptr := (*C.uint8_t)(unsafe.Pointer(&data[0]))
	var w, h C.uint32_t
	ret := C.rivex_image_dimensions(ptr, C.size_t(len(data)), &w, &h)
	if ret == 0 {
		return 0, 0, false
	}
	return uint32(w), uint32(h), true
}

// PDFExtractText extracts text from PDF bytes.
func PDFExtractText(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	ptr := (*C.uint8_t)(unsafe.Pointer(&data[0]))
	result := C.rivex_pdf_extract_text(ptr, C.size_t(len(data)))
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}
