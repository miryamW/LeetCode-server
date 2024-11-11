# Step 1: Build the application
FROM golang:1.23-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy files into the container
COPY . .

# Install dependencies
RUN go mod tidy

# Build the application
RUN go build -o backend .

# Run the backend
CMD ["go", "run", "main.go"]
