# Build stage — runs on the build host's native arch and cross-compiles,
# so multi-arch CI builds don't pay the QEMU tax for the Go compile.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

# hadolint ignore=DL3018
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY backend/ ./backend/

# TARGETOS / TARGETARCH MUST be declared bare (no defaults): buildx
# supplies them as global automatic args, and a stage-level default
# shadows the supplied value (see team-metrics' Graviton post-mortem).
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o /app/focus \
    ./backend/cmd/focus

# Final stage
FROM alpine:3.21

# hadolint ignore=DL3018
RUN apk --no-cache add ca-certificates tzdata

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

COPY --from=builder /app/focus /app/focus
COPY backend/migrations /app/migrations

RUN chown -R appuser:appuser /app
USER appuser

EXPOSE 8000

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8000/health || exit 1

ENTRYPOINT ["/app/focus"]
CMD ["server"]
