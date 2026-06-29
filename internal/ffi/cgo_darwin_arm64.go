//go:build darwin && arm64

package ffi

// #cgo LDFLAGS: -L${SRCDIR}/../../lib/darwin_arm64 -lrivex_ffi -framework CoreFoundation -liconv
import "C"
