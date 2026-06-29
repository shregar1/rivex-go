//go:build linux && amd64

package ffi

// #cgo LDFLAGS: -L${SRCDIR}/../../lib/linux_amd64 -lrivex_ffi -lm -ldl -lpthread
import "C"
