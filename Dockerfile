# Stage 1: The builder
FROM golang:1.22-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -o /willitrain .

# Stage 2: The final image
FROM alpine:latest

WORKDIR /app

# Install the timezone database
RUN apk --no-cache add tzdata

# Copy the built binary from the builder stage
COPY --from=builder /willitrain /willitrain

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["/willitrain"]
