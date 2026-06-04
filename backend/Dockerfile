# ------------------------------------------------------------
# Stage 1 — Build the Go binary
# ------------------------------------------------------------
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build tools
RUN apk add --no-cache git

# Copy go.mod and go.sum first (better caching)
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy the rest of the backend source
COPY backend/ .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o step-ui-backend ./cmd

# ------------------------------------------------------------
# Stage 2 — Minimal runtime image
# ------------------------------------------------------------
FROM alpine:3.19

WORKDIR /app

# Add CA certificates (needed for HTTPS calls to step-ca)
RUN apk add --no-cache ca-certificates

# Copy compiled binary
COPY --from=builder /app/step-ui-backend .

# Expose backend port
EXPOSE 8080

# Run backend
CMD ["./step-ui-backend"]
