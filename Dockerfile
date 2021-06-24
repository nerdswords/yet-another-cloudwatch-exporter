FROM golang:1.15 as builder

WORKDIR /opt/

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN go test -cover ./...

ENV GOOS linux
ARG GOARCH
ENV GOARCH ${GOARCH:-amd64}
ENV CGO_ENABLED=0

ARG VERSION
RUN go build -v -ldflags "-X main.version=$VERSION" -o yace cmd/yace/main.go

FROM alpine:latest

EXPOSE 5000
ENTRYPOINT ["yace"]
CMD ["--config.file=/tmp/config.yml"]
RUN addgroup -g 1000 exporter && \
    adduser -u 1000 -D -G exporter exporter -h /exporter

WORKDIR /exporter/


RUN apk --no-cache add ca-certificates
COPY --from=builder /opt/yace /usr/local/bin/yace
USER exporter

