# backend/Dockerfile
# build stage
FROM golang:1.22-alpine AS builder
WORKDIR /src

# system deps for some modules (optional)
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# adjust the build target path if your main package is elsewhere (e.g., ./cmd/server)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/step-ui-backend ./cmd

# final stage
FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/step-ui-backend /usr/local/bin/step-ui-backend

# create directory for sqlite DB and drop privileges
RUN mkdir -p /var/lib/step-ui && chown -R 1000:1000 /var/lib/step-ui
USER 1000
WORKDIR /var/lib/step-ui

ENV DATABASE_DSN="file:/var/lib/step-ui/dev.db?_foreign_keys=1"
EXPOSE 8080

CMD ["/usr/local/bin/step-ui-backend"]
