# ============================
# 1) Build stage
# ============================
FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o step-ui-backend ./cmd/server

# ============================
# 2) Runtime stage
# ============================
FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/step-ui-backend /usr/local/bin/step-ui-backend

EXPOSE 8080

CMD ["step-ui-backend"]
