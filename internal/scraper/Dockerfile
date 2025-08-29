# --- Builder Stage ---
# This stage compiles the Go application.
FROM golang:1.24-alpine AS builder

# Set the working directory inside the container.
WORKDIR /app

# Copy go.mod and go.sum files to download dependencies first.
# This leverages Docker's layer caching.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code.
COPY . .

# Build the scraper application.
# -o /scraper specifies the output binary name.
# CGO_ENABLED=0 and GOOS=linux ensure a statically linked binary for a linux environment.
RUN CGO_ENABLED=0 GOOS=linux go build -o /scraper .

# --- Final Stage ---
# This stage creates the final, small, and secure image.
FROM alpine:latest

# Copy the compiled binary from the builder stage.
COPY --from=builder /scraper /scraper

# Expose the port the server will listen on.
EXPOSE 8080

# Set the command to run when the container starts.
CMD ["/scraper"]
