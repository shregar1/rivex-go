//go:build windows && amd64

package ffi

// #cgo LDFLAGS: -L${SRCDIR}/../../lib/windows_amd64 -lrivex_ffi -lws2_32 -lbcrypt -luserenv -lntdll -lkernel32 -ladvapi32
import "C"
