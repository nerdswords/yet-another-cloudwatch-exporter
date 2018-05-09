FROM golang:1.10

RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

WORKDIR /go/src/app
ADD ./src/Gopkg.lock ./
ADD ./src/Gopkg.toml ./
RUN dep ensure -vendor-only

ENV GOOS darwin
ENV GOARCH amd64
RUN go build -v -o yace-$GOOS-$GOARCH
Add ./src/ ./
ENV GOOS linux
ENV GOARCH amd64
RUN go build -v -o yace-$GOOS-$GOARCH
ENV CGO_ENABLED=0
RUN go build -v -o yace-alpine-$GOARCH

FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /go/src/app/yace-linux-amd64 .
COPY --from=0 /go/src/app/yace-darwin-amd64 .
COPY --from=0 /go/src/app/yace-alpine-amd64 /usr/local/bin/yace
CMD ["./yace-linux-amd64"]
