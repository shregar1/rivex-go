//go:build linux && arm64

package ffi

// #cgo LDFLAGS: -L${SRCDIR}/../../lib/linux_arm64 -lrivex_ffi -lm -ldl -lpthread
import "C"
