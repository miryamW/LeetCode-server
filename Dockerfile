############################
# STEP 1 build executable binary
############################
FROM golang:alpine AS builder

# Install git and CA certificates
RUN apk update && apk add --no-cache git ca-certificates

# Set environment variable for Go modules
ENV GO111MODULE=on

# Set the working directory inside the container
WORKDIR $GOPATH/src/packages/goginapp/
COPY . .

# Fetch dependencies
RUN go mod download

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/main .

############################
# STEP 2 build a small image
############################
FROM alpine:3

# Install CA certificates
RUN apk update && apk add --no-cache ca-certificates

# Set the working directory
WORKDIR /

# Copy the static executable from the builder stage
COPY --from=builder /go/main /go/main
#COPY public /go/public

# Set environment variables
ENV PORT=8080
ENV GIN_MODE=release

# Expose port 8080 to the outside world
EXPOSE 8080

# Set the working directory
WORKDIR /go

# Run the Go Gin binary
ENTRYPOINT ["/go/main"]
