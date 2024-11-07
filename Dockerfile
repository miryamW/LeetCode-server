# Step 1: Build the application
FROM golang:1.22.2 AS builder

# Update the package list and install CA certificates
RUN apt-get update && apt-get install -y ca-certificates

# Set the working directory inside the container
WORKDIR /go/src/leetcode-server

# Copy the source code into the container
COPY . .

# Update Go modules before building
RUN go mod tidy

# Build the application
RUN go build -o myserver main.go

# Step 2: Create the final image with the binary
FROM debian:bullseye-slim

# Copy the binary from the builder stage to the final image
COPY --from=builder /go/src/leetcode-server/myserver /usr/local/bin/myserver

# Expose the port the server will run on
EXPOSE 8080

# Run the binary as the container's entrypoint
CMD ["/usr/local/bin/myserver"]
