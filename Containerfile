# Use a lightweight Go base image
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o msgqueue .

# Final stage
FROM alpine:latest

# Install ca-certificates for any HTTPS needs (optional)
RUN apk --no-cache add ca-certificates

# Create a non-root user
RUN adduser -D appuser

# Set working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/msgqueue .

# Copy init.sql for schema
COPY --from=builder /app/init.sql .

# Copy ui.html
COPY --from=builder /app/ui.html .

# Change ownership
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Create data directory if needed
RUN mkdir -p /app/data

# Run the application
CMD ["./msgqueue"]