# ╔══════════════════════════════════════════════════════════════════════════╗
# ║  BongCloud Chess — Multi-stage Dockerfile                               ║
# ║  Stage 1: deps    — download and cache Go modules                       ║
# ║  Stage 2: builder — compile the binary                                  ║
# ║  Stage 3: runner  — minimal distroless runtime image                    ║
# ╚══════════════════════════════════════════════════════════════════════════╝

# Stage 1: Module cache 
FROM golang:1.23-alpine AS deps

WORKDIR /app

# Copy dependency manifests first to leverage Docker layer caching.
# Modules are only re-downloaded when go.mod / go.sum change.
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Stage 2: Builder
FROM deps AS builder

WORKDIR /app

# Copy the rest of the source
COPY . .

# Build flags:
#   -trimpath     strips module path prefixes for reproducible builds
#   -ldflags      removes debug info / DWARF tables, injects version
#   CGO_ENABLED=0 produces a fully static binary (no libc dependency)
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /bin/bongcloud \
    ./cmd/server

#  Stage 3: Runner
FROM gcr.io/distroless/static-debian12 AS runner

# Copy only the compiled binary — no shell, no package manager, minimal attack surface
COPY --from=builder /bin/bongcloud /bin/bongcloud

# Run as non-root user (distroless uid 65532 = "nonroot")
USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/bin/bongcloud"]