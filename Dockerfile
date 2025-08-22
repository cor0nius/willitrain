# Stage 1: Frontend builder
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

# Copy frontend package files
COPY frontend/package*.json ./

# Install frontend dependencies
RUN npm install

# Copy the rest of the frontend source code
COPY frontend/ ./

# Build the frontend
RUN npm run build

# Stage 2: Go builder
FROM golang:1.24.4-bookworm AS go-builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the Go source code
COPY . .

# Copy the built frontend from the frontend-builder stage
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -o /willitrain .

# Stage 3: The final image
FROM alpine:latest

WORKDIR /app

# Install the timezone database
RUN apk --no-cache add tzdata

# Copy the built binary from the go-builder stage
COPY --from=go-builder /willitrain /willitrain

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["/willitrain"]
