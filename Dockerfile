# Step 1: Use Golang base image
FROM golang:1.23-alpine as builder

# Step 3: Set up working directory
WORKDIR /app

# Step 4: Copy application code to container
COPY . .

# Step 5: Install Go dependencies
RUN go mod tidy

# Step 6: Build the Go application
RUN go build -o backend .

# Step 8: Define the command to run your application
CMD ["go", "run", "main.go"]
