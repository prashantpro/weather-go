# Use a multi-stage build for efficient image size

# Stage 1: Build the Go application
FROM golang:1.22-alpine AS build
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code and build the application
COPY . .
RUN go build -o weather-api

# Stage 2: Run the Go application
FROM alpine:3.18
WORKDIR /app

# Copy the compiled binary from the build stage
COPY --from=build /app/weather-api /app/weather-api

# Expose the application port
EXPOSE 8080

# Run the application
ENTRYPOINT ["/app/weather-api"]