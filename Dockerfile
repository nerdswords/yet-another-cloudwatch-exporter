FROM golang:1.10

WORKDIR /go/src/app
ADD ./ ./
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
RUN dep ensure
ENV GOOS darwin
ENV GOARCH amd64
RUN go build -v -o yace-$GOOS-$GOARCH
ENV GOOS linux
ENV GOARCH amd64
RUN go build -v -o yace-$GOOS-$GOARCH


FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /go/src/app/yace-linux-amd64 .
COPY --from=0 /go/src/app/yace-darwin-amd64 .
CMD ["./yace-linux-amd64"]
