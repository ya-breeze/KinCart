# Build stage
FROM golang:1.25-alpine3.21 AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Pre-fetch dependencies using cache mount
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source and build using cache mounts
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 go build -tags musl -ldflags="-s -w" -o kincart-server cmd/server/main.go && \
    CGO_ENABLED=1 go build -tags musl -ldflags="-s -w" -o kincart-admin cmd/admin/main.go

# Run stage
FROM alpine:3.21

WORKDIR /app

# Add a non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/kincart-server .
COPY --from=builder /app/kincart-admin .

# Ensure appuser owns the app directory
RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

CMD ["./kincart-server"]
