.DEFAULT_GOAL := build

build:
	go build -o yace cmd/yace/main.go

test:
	go test -v -race -count=1 ./...

lint:
	golangci-lint run -v -c .golangci.yml
