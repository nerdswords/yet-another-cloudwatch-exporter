FROM golang:1.12 as builder

WORKDIR /opt/

COPY go.mod go.sum ./
RUN go mod download

Add ./*.go ./
RUN go test

ENV GOOS linux
ENV GOARCH amd64
ENV CGO_ENABLED=0

ARG VERSION
RUN go build -v -ldflags "-X main.version=$VERSION" -o yace

FROM alpine:latest

EXPOSE 5000
ENTRYPOINT ["yace"]
CMD ["--config.file=/tmp/config.yml"]
WORKDIR /root/

RUN apk --no-cache add ca-certificates
COPY --from=builder /opt/yace /usr/local/bin/yace

