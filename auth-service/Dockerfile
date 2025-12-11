FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /auth-service main.go

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy binary from builder
COPY --from=builder /auth-service .
COPY config/config.json ./config/

EXPOSE 3001

CMD ["./auth-service"]
