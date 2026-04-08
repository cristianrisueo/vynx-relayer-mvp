# ──────────────────────────────────────────────────────────────────────────────
# Stage 1: Builder
# Compiles the Go binary in a full SDK environment with CGO support.
# go-ethereum requires CGO for its C-KZG cryptographic bindings.
# ──────────────────────────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

# Install C toolchain required by go-ethereum's CGO dependencies.
RUN apk add --no-cache gcc musl-dev ca-certificates git

WORKDIR /build

# Cache dependency resolution as a separate layer.
COPY go.mod go.sum ./
RUN go mod download

# Copy the full source tree and compile the relayer binary.
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /relayer \
    ./cmd/relayer

# ──────────────────────────────────────────────────────────────────────────────
# Stage 2: Production image
# Minimal Alpine image containing only the compiled binary and TLS certificates.
# ──────────────────────────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

# Run as a non-root user for security hardening.
RUN adduser -D -u 1001 relayer
USER relayer

COPY --from=builder /relayer /relayer

EXPOSE 8080

ENTRYPOINT ["/relayer"]
