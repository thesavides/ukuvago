# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy source code
COPY . .

# Generate go.sum and download dependencies
RUN go mod tidy

# Build the application (CGO disabled for simpler build)
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Copy binary from builder
COPY --from=builder /app/server .

# Copy web files
COPY --from=builder /app/web ./web

# Create uploads directory
RUN mkdir -p uploads

# Expose port
EXPOSE 8080

# Run the application
CMD ["./server"]
