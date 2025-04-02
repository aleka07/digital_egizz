# Stage 1: Build the Go application
FROM golang:1.21-alpine AS builder
# (User can update Go version if needed, e.g., 1.22)

WORKDIR /app

# Copy go mod and sum files first to leverage Docker cache
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY *.go ./

# Build the application statically (recommended for scratch/alpine images)
# CGO_ENABLED=0 is important for static linking without C libraries
# -ldflags="-w -s" reduces binary size
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/server .

# Stage 2: Create the final lightweight image
FROM alpine:latest

WORKDIR /app

# Copy only the built binary from the builder stage
COPY --from=builder /app/server /app/server

# Expose the port the application runs on (for documentation, not strictly needed by Docker networking)
EXPOSE 8081

# Command to run the executable
CMD ["/app/server"] 