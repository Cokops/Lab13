FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY orchestrator/ ./orchestrator/

# Build the orchestrator
RUN CGO_ENABLED=0 GOOS=linux go build -o orchestrator orchestrator/main.go

# Multi-stage: minimal runtime image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/orchestrator .

# Expose API port
EXPOSE 8080

# Command to run the executable
CMD ["./orchestrator"]