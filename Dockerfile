FROM golang:1.15-alpine3.12 as builder

COPY main.go /go/src/js-demo/main.go
WORKDIR /go/src/js-demo
RUN go env -w GOPROXY=https://goproxy.cn,direct && \
    go mod init && \
    go mod download
# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o js-demo main.go

FROM alpine:3.12
WORKDIR /
COPY --from=builder /go/src/js-demo/js-demo .
ENV NATS_URL="" ROLE="edge" SUBJECT="" STREAM="" CONSUMER=""
CMD ["./js-demo"]