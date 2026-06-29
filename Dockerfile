# ---- Stage 1: Build the example application ----
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

# Copy module files first for better layer caching
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true

# Copy the full source tree
COPY . .

# Build the hello example as a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app ./examples/hello

# ---- Stage 2: Minimal runtime image ----
FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app /app/server

EXPOSE 8000

ENTRYPOINT ["/app/server"]
