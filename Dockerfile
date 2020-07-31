FROM golang:1.14 as builder

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

ENV AWS_REGION eu-west-1
ENV CONFIG_FILE /tmp/config.yml

ENTRYPOINT ["yace"]
CMD ["--config.file=${CONFIG_FILE}"]
RUN addgroup -g 1000 exporter && \
    adduser -u 1000 -D -G exporter exporter -h /exporter

WORKDIR /exporter/


RUN apk --no-cache add ca-certificates
COPY --from=builder /opt/yace /usr/local/bin/yace
USER exporter

