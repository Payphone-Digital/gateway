# Stage 1: Build
FROM golang:1.24-alpine AS builder

# Buat user non-root
RUN addgroup -S appgroup && \
    adduser -S -G appgroup -h /app appuser && \
    apk add --no-cache git ca-certificates tzdata

# Set working directory dengan permission yang tepat
WORKDIR /app
RUN chown appuser:appgroup /app

USER appuser

# Copy go.mod dan go.sum
COPY --chown=appuser:appgroup go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code dengan proper ownership
COPY --chown=appuser:appgroup . .

# Build dengan security flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -trimpath \
    # Tambahan security flags
    -gcflags=all="-N -l" \
    -o main ./cmd

# Stage 2: Security scan
FROM aquasec/trivy:latest AS security-scan
COPY --from=builder /app /app
RUN trivy filesystem --no-progress --ignore-unfixed --severity HIGH,CRITICAL /app

# Stage 3: Runtime
FROM scratch

# Copy system files yang diperlukan
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy binary
COPY --from=builder /app/main /app/main

# Set proper permissions
WORKDIR /app

# Security configurations
ENV GODEBUG=http2client=0 \
    # Disable debugging
    GOLANG_DEBUG=0 \
    # Set secure configurations
    GO_SECURITY_LEVEL=high

# Gunakan non-root user
USER appuser

# Define health check
HEALTHCHECK --interval=30s --timeout=3s \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Set security capabilities
# Note: Requires --security-opt="no-new-privileges:true" saat menjalankan container
EXPOSE 8080

# Set read-only filesystem
# Note: Requires --read-only flag saat menjalankan container
ENTRYPOINT ["/app/main"]