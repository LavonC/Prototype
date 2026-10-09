# Build Stage
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build the coordinator binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/coordinator ./cmd/coordinator

# Runtime Stage
FROM alpine:3.19

WORKDIR /app

# Install CA certificates for TLS
RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/bin/coordinator /usr/local/bin/coordinator
COPY --from=builder /app/internal/storage/migrations /app/migrations

# Expose HTTP/WebSocket port
EXPOSE 8080

ENTRYPOINT ["coordinator"]
