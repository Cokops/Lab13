FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY agents/collector/ ./agents/collector/

# Build the collector agent
RUN CGO_ENABLED=0 GOOS=linux go build -o collector agents/collector/main.go

# Multi-stage: minimal runtime image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/collector .

# Expose any ports if needed
# EXPOSE 8080

# Command to run the executable
CMD ["./collector"]