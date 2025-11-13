# Build stage
FROM golang:1.24-alpine3.22 AS builder

WORKDIR /app

# Install build dependencies (Node.js for frontend, git for Go)
RUN apk add --no-cache git nodejs npm

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy all source files (Railway builds from repo root)
COPY . .

# Build frontend
WORKDIR /app/frontend
RUN npm install && npm run build

# Build the application
WORKDIR /app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Copy frontend build output
COPY --from=builder /app/frontend/out ./frontend/out

# Expose port
EXPOSE 8080

# Run the server
CMD ["./server"]

