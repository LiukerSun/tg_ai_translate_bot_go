# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o tg-bot-go main.go

# Run stage
FROM alpine:latest

# Set timezone
ENV TZ=Asia/Shanghai

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata && \
    update-ca-certificates

WORKDIR /app

# Create logs directory
RUN mkdir -p /app/logs

# Copy binary and config from builder
COPY --from=builder /app/tg-bot-go .
COPY --from=builder /app/config ./config

# Ensure binary is executable
RUN chmod +x /app/tg-bot-go

# Run the application
ENTRYPOINT ["./tg-bot-go"]