# ============================
# 1) Build stage
# ============================
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy Go module files
COPY go.mod go.sum ./
RUN go mod download

# Copy backend source
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o step-ui-backend .

# ============================
# 2) Runtime stage
# ============================
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/step-ui-backend .

EXPOSE 8080
ENV PORT=8080

CMD ["./step-ui-backend"]
