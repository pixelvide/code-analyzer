# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go module files
COPY go.mod go.sum ./
# Download dependencies
RUN go mod download

# Copy source code
COPY main.go ./
COPY analyzers/ ./analyzers/
COPY config/ ./config/
COPY models/ ./models/
COPY utils/ ./utils/

# Build the binary
RUN go build -o code-analyzer .

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/code-analyzer .
# Copy default config
COPY analysis-config.yaml .

# Set entrypoint
ENTRYPOINT ["./code-analyzer"]
