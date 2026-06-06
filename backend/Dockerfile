# ============================
# 1) Build stage
# ============================
FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates

# Copy go.mod and go.sum first
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build the backend (main.go is in backend/cmd)
RUN CGO_ENABLED=0 GOOS=linux go build -o step-ui-backend ./cmd

# ============================
# 2) Runtime stage
# ============================
FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/step-ui-backend /usr/local/bin/step-ui-backend

EXPOSE 8080

CMD ["step-ui-backend"]
