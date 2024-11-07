# Use a Go base image to build the Go server
FROM golang:1.20 AS build

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files and download the dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go application
RUN go build -o server .

# Create a minimal image to run the Go server
FROM debian:bullseye-slim

# Install necessary dependencies (for example, curl if needed)
RUN apt-get update && apt-get install -y curl

# Set the working directory inside the container
WORKDIR /app

# Copy the built Go server from the build stage
COPY go.mod go.sum ./

# Expose the port the server will run on
EXPOSE 8080

# Run the Go server
CMD ["./server"]
