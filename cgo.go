package rivex

// Shared cgo compiler flags for the rivex package. The C ABI header for the Rust
// core lives in include/; the platform-specific linker flags (which static
// library to link and the system libraries the Rust runtime needs) are supplied
// by the build-constrained cgo_<goos>_<goarch>.go files in this directory.

// #cgo CFLAGS: -I${SRCDIR}/include
import "C"
