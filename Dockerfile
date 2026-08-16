# Stage 1: Build the Go binary
FROM golang:1.26.2-alpine AS builder

WORKDIR /app

# Install git and ca-certificates for secure HTTPS requests if needed
RUN apk add --no-cache git ca-certificates

# Copy go.mod and go.sum and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go application with optimizations (disable CGO, strip debug symbols and DWARF tables)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/api/main.go

# Stage 2: Final minimal and secure image
FROM scratch

# Copy CA certificates for SSL/TLS connections
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the statically-linked binary from the builder
COPY --from=builder /server /server

# Expose the application port
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/server"]
