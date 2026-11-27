# Rivex Go SDK Makefile
#
# Prerequisites:
#   - Go 1.22+
#   - C toolchain (clang/gcc) for cgo
#   - Rust toolchain (only when rebuilding librivex_ffi from source)
#
# The Rust core is vendored as prebuilt static libraries under lib/.
# To rebuild them from source, clone rivex-core as a sibling directory
# and run: make vendor-libs

GO ?= go
GOHOSTOS ?= $(shell $(GO) env GOHOSTOS)
GOHOSTARCH ?= $(shell $(GO) env GOHOSTARCH)

.PHONY: all build build-all test test-race lint fmt clean vendor-libs

all: fmt lint build test

build:
	$(GO) build ./admin/... ./internal/ffi/...

build-all:
	$(GO) build ./...

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

test-v:
	$(GO) test -v ./...

lint:
	$(GO) vet ./admin/... ./internal/ffi/...

fmt:
	$(GO) fmt ./...

# Rebuild librivex_ffi from the sibling rivex-core repo and copy it
# into the vendored lib/ directory for the host platform.
vendor-libs:
	@if [ ! -d ../core.rivex.vexarr.com ]; then \
		echo "Error: ../core.rivex.vexarr.com not found"; \
		echo "Clone rivex-core as a sibling directory:"; \
		echo "  git clone https://github.com/shregar1/rivex-core.git ../core.rivex.vexarr.com"; \
		exit 1; \
	fi
	cargo build -p rivex-ffi --release --manifest-path ../core.rivex.vexarr.com/Cargo.toml
	mkdir -p lib/$(GOHOSTOS)_$(GOHOSTARCH)
	cp ../core.rivex.vexarr.com/target/release/librivex_ffi.a lib/$(GOHOSTOS)_$(GOHOSTARCH)/
	cp ../core.rivex.vexarr.com/crates/rivex-ffi/rivex_ffi.h include/rivex_ffi.h
	@echo "Vendored librivex_ffi for $(GOHOSTOS)/$(GOHOSTARCH)"

clean:
	$(GO) clean ./...
	rm -rf lib/*/librivex_ffi.a
