FROM golang:1.23-alpine as builder

RUN apk add --no-cache curl bash \
    && curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" \
    && chmod +x kubectl \
    && mv kubectl /usr/local/bin/

WORKDIR /app

COPY . .

RUN go mod tidy

RUN go build -o backend .

CMD ["go", "run", "main.go"]
