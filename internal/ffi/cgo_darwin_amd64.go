//go:build darwin && amd64

package ffi

// #cgo LDFLAGS: -L${SRCDIR}/../../lib/darwin_amd64 -lrivex_ffi -framework CoreFoundation -liconv
import "C"
