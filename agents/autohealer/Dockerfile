FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY agents/autohealer/ ./agents/autohealer/

# Build the autohealer agent
RUN CGO_ENABLED=0 GOOS=linux go build -o autohealer agents/autohealer/main.go

# Multi-stage: minimal runtime image
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/autohealer .

# Command to run the executable
CMD ["./autohealer"]