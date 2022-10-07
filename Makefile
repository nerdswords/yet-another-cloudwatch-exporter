.DEFAULT_GOAL := build

build:
	go build -v -o yace ./cmd/yace

test:
	go test -v -race -count=1 ./...

lint:
	golangci-lint run -v -c .golangci.yml
