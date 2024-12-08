# Stage 1: Build the Go application
FROM docker.arvancloud.ir/golang:1.22-alpine

WORKDIR /app

COPY . .

RUN go env -w GOPROXY=https://goproxy.cn,direct

RUN apk add build-base gcc git
RUN apk add vips-dev

#RUN go mod download
RUN go mod tidy
RUN go build -o app main.go


RUN apk add sqlite


EXPOSE 8080

CMD ["sh", "-c", "./app"]
