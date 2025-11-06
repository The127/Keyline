# ---- Build stage ----
FROM golang:1.25 AS builder

# Set up work dir
WORKDIR /app

# Enable Go modules
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

# Cache dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binary
RUN go build -o api ./cmd/api

# ---- Runtime stage ----
FROM alpine:3.22.1 AS runtime

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/api /app/api

# Expose port (adjust if needed)
EXPOSE 8080

# Run the server
CMD ["/app/api"]
