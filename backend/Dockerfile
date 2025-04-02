FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod ./
# Only copy go.sum if it exists
COPY go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /digital-egiz-backend ./cmd

# Use a small alpine image for the final container
FROM alpine:3.19

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /digital-egiz-backend .

# Expose the port the server listens on
EXPOSE 8080

# Run the binary
CMD ["./digital-egiz-backend"] 