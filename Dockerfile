# Stage 1: Build the Go binary
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod ./
RUN go mod download

# Copy source code files
COPY . .

# Build a statically linked Linux binary
RUN CGO_ENABLED=0 GOOS=linux go build -o fileshare main.go

# Stage 2: Create a tiny runtime image
FROM alpine:latest

WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app/fileshare .

# Copy the web folder containing index.html, style.css, and app.js
COPY --from=builder /app/web ./web

# Expose port 8080
EXPOSE 8080
    
# Run the relay server automatically on port 8080
CMD ["./fileshare", "server", "-web"]