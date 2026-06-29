package rivex

/*
#include "rivex_ffi.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"

// ReMatch tests whether a regex pattern matches anywhere in the text.
//
// The pattern is compiled and matched in a single call -- for repeated use of
// the same pattern, consider compiling separately. Uses the Rust regex engine,
// which provides linear-time matching guarantees.
//
// Returns false if the pattern is invalid or does not match.
func ReMatch(pattern, text string) bool {
	cPattern := C.CString(pattern)
	defer C.free(unsafe.Pointer(cPattern))
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))

	return C.rivex_re_match(cPattern, cText) == 1
}
