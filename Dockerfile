FROM golang:1.12 as builder

RUN curl -fsSL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.5.1/dep-linux-amd64 && chmod +x /usr/local/bin/dep

WORKDIR /opt/

ADD go.mod ./go.sum ./
Add ./*.go ./

RUN go test

ENV GOOS linux
ENV GOARCH amd64
ENV CGO_ENABLED=0

ARG VERSION
RUN go build -v -ldflags "-X main.version=$VERSION" -o yace

FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /opt/yace /usr/local/bin/yace
CMD ["/usr/local/bin/yace"]
