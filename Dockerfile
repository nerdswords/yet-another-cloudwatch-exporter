FROM golang:1.20-alpine as builder
LABEL maintainer="YACE developers"
RUN apk update && apk add --no-cache make ca-certificates && update-ca-certificates

ENV USER=yace
ENV UID=10001
ENV GOOS linux
ENV CGO_ENABLED=0

RUN adduser --disabled-password --gecos "" --home "/nonexistent" --shell "/sbin/nologin" --no-create-home --uid "${UID}" "${USER}"

WORKDIR /opt/

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

ARG VERSION
RUN make 


FROM scratch
LABEL maintainer="YACE developers"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /opt/yace /usr/local/bin/yace

WORKDIR /exporter/
USER yace:yace

EXPOSE 5000
ENTRYPOINT ["yace"]
CMD ["--config.file=/tmp/config.yml"]
