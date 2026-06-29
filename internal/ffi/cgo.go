package ffi

// Shared cgo compiler flags for the ffi package. Platform-specific linker flags
// are supplied by the build-constrained cgo_<goos>_<goarch>.go files here.

// #cgo CFLAGS: -I${SRCDIR}/../../include
import "C"
