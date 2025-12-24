# Multi-stage build for Go application
FROM golang:1.25.5-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod and sum files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the application
COPY cmd ./cmd
COPY internal ./internal
COPY web ./web

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

# Final stage - lightweight image
FROM alpine:3.19

# Install ca-certificates for HTTPS support
RUN apk --no-cache add ca-certificates

# Create non-root user for security
RUN addgroup -g 1000 app && adduser -u 1000 -G app -s /bin/sh -D app

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/main .
COPY --from=builder /app/web ./web

# Create data directory for SQLite database
RUN mkdir -p /app/data

# Change ownership to non-root user
RUN chown -R app:app /app

# Switch to non-root user
USER app

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]
