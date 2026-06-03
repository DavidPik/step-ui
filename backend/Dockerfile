# ============================
# 1) Build stage
# ============================
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Go mod cache
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy source
COPY backend/ .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o step-ui-backend .

# ============================
# 2) Runtime stage
# ============================
FROM alpine:3.19

WORKDIR /app

# Copy binary
COPY --from=builder /app/step-ui-backend .

# Expose backend port
EXPOSE 8080

# Environment variables (can be overridden in compose)
ENV PORT=8080

CMD ["./step-ui-backend"]
