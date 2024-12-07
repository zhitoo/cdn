# Stage 1: Build the Go application
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY . .

# RUN go env -w GOPROXY=https://goproxy.cn,direct

#RUN go mod download
#RUN go mod verify
RUN go build -o app main.go

# Stage 2: Run the Go application
FROM alpine:latest

RUN apk add --no-cache build-base gcc git
RUN apk add --no-cache vips-dev

WORKDIR /root/

COPY --from=builder /app/app .
COPY --from=builder /app/public ./public

EXPOSE 8080

CMD ["./app"]
