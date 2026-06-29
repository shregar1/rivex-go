#!/usr/bin/env bash
# Build the rivex-ffi static library from the sibling Rust core and vendor it
# (plus the C header) into this Go module so `go get` / `go build` is fully
# self-contained — no external Rust toolchain or library fetch at consume time.
#
# Go modules are source-only, so the prebuilt static archives MUST be committed
# to this repo. CI runs this on each target OS/arch and commits the result.
#
# Usage:
#   scripts/vendor-libs.sh                 # build for the host GOOS/GOARCH
#   scripts/vendor-libs.sh <goos> <goarch> <rust-target>
#
# Examples:
#   scripts/vendor-libs.sh darwin arm64 aarch64-apple-darwin
#   scripts/vendor-libs.sh linux  amd64 x86_64-unknown-linux-gnu
#   scripts/vendor-libs.sh linux  arm64 aarch64-unknown-linux-gnu
#   scripts/vendor-libs.sh windows amd64 x86_64-pc-windows-gnu
set -euo pipefail

REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CORE_DIR="${RIVEX_CORE_DIR:-$REPO_DIR/../core.rivex.vexarr.com}"

GOOS="${1:-$(go env GOOS)}"
GOARCH="${2:-$(go env GOARCH)}"
RUST_TARGET="${3:-}"

if [[ ! -d "$CORE_DIR" ]]; then
  echo "error: Rust core not found at $CORE_DIR (set RIVEX_CORE_DIR)" >&2
  exit 1
fi

EXTRA=()
if [[ -n "$RUST_TARGET" ]]; then
  EXTRA=(--target "$RUST_TARGET")
  rustup target add "$RUST_TARGET" >/dev/null 2>&1 || true
  BUILT="$CORE_DIR/target/$RUST_TARGET/release/librivex_ffi.a"
else
  BUILT="$CORE_DIR/target/release/librivex_ffi.a"
fi

echo ">> building rivex-ffi (release) for ${GOOS}/${GOARCH} ${RUST_TARGET:+($RUST_TARGET)}"
( cd "$CORE_DIR" && cargo build --release -p rivex-ffi "${EXTRA[@]}" )

DEST="$REPO_DIR/lib/${GOOS}_${GOARCH}"
mkdir -p "$DEST"
cp "$BUILT" "$DEST/librivex_ffi.a"
cp "$CORE_DIR/crates/rivex-ffi/include/rivex_ffi.h" "$REPO_DIR/include/rivex_ffi.h"

echo ">> vendored:"
echo "   $DEST/librivex_ffi.a"
echo "   $REPO_DIR/include/rivex_ffi.h"
