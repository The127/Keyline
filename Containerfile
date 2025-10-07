# ---- Build stage ----
FROM golang:1.24 AS builder

WORKDIR /app

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build minimal static binary
RUN go build -ldflags="-s -w" -o /keyline ./cmd/api

# ---- Runtime stage ----
FROM gcr.io/distroless/static-debian12
COPY --from=builder /keyline /
USER nonroot:nonroot
ENTRYPOINT ["/keyline"]
