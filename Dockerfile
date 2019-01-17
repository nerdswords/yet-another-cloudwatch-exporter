FROM golang:1.10 as builder

RUN curl -fsSL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.4.1/dep-linux-amd64 && chmod +x /usr/local/bin/dep

WORKDIR /go/src/yace
ADD ./src/Gopkg.lock ./src/Gopkg.toml ./
RUN dep ensure -vendor-only

Add ./src/ ./

RUN go test

ENV GOOS linux
ENV GOARCH amd64
ENV CGO_ENABLED=0
RUN go build -v

FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/yace/yace /usr/local/bin/yace
CMD ["/usr/local/bin/yace"]
