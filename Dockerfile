FROM golang:1.23-alpine as builder   
    
WORKDIR /app

COPY . .

RUN go mod tidy

RUN go build -o backend .


CMD ["go", "run", "main.go"]