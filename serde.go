package rivex

/*
#include "rivex_ffi.h"
#include <stdlib.h>
*/
import "C"
import (
	"unsafe"
)

// YAMLParse parses a YAML string and returns the equivalent JSON string.
func YAMLParse(yaml string) string {
	cYAML := C.CString(yaml)
	defer C.free(unsafe.Pointer(cYAML))
	result := C.rivex_yaml_parse(cYAML)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// YAMLDump converts a JSON string to a YAML string.
func YAMLDump(json string) string {
	cJSON := C.CString(json)
	defer C.free(unsafe.Pointer(cJSON))
	result := C.rivex_yaml_dump(cJSON)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// TOMLParse parses a TOML string and returns the equivalent JSON string.
func TOMLParse(tomlStr string) string {
	cTOML := C.CString(tomlStr)
	defer C.free(unsafe.Pointer(cTOML))
	result := C.rivex_toml_parse(cTOML)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// TOMLDump converts a JSON string to a TOML string.
func TOMLDump(json string) string {
	cJSON := C.CString(json)
	defer C.free(unsafe.Pointer(cJSON))
	result := C.rivex_toml_dump(cJSON)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// CSVParse parses a CSV string and returns a JSON array-of-arrays string.
func CSVParse(csvStr string) string {
	cCSV := C.CString(csvStr)
	defer C.free(unsafe.Pointer(cCSV))
	result := C.rivex_csv_parse(cCSV)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// CSVDump converts a JSON array-of-arrays string to CSV.
func CSVDump(json string) string {
	cJSON := C.CString(json)
	defer C.free(unsafe.Pointer(cJSON))
	result := C.rivex_csv_dump(cJSON)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// MsgpackEncode encodes a JSON string as MessagePack bytes.
func MsgpackEncode(jsonStr string) []byte {
	cJSON := C.CString(jsonStr)
	defer C.free(unsafe.Pointer(cJSON))
	var outLen C.size_t
	result := C.rivex_msgpack_encode(cJSON, &outLen)
	if result == nil {
		return nil
	}
	defer C.rivex_bytes_free(result, outLen)
	return C.GoBytes(unsafe.Pointer(result), C.int(outLen))
}

// MsgpackDecode decodes MessagePack bytes to a JSON string.
func MsgpackDecode(data []byte) string {
	var ptr *C.uint8_t
	if len(data) > 0 {
		ptr = (*C.uint8_t)(unsafe.Pointer(&data[0]))
	}
	result := C.rivex_msgpack_decode(ptr, C.size_t(len(data)))
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// XMLToJSON converts an XML string to a JSON string.
func XMLToJSON(xml string) string {
	cXML := C.CString(xml)
	defer C.free(unsafe.Pointer(cXML))
	result := C.rivex_xml_to_json(cXML)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}

// JSONToXML converts a JSON string to an XML string.
func JSONToXML(json string) string {
	cJSON := C.CString(json)
	defer C.free(unsafe.Pointer(cJSON))
	result := C.rivex_json_to_xml(cJSON)
	if result == nil {
		return ""
	}
	defer C.rivex_free_string(result)
	return C.GoString(result)
}
