.PHONY: all

all: check build

check:
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.42.1
	@golangci-lint run

build:
	@go build -o cmd/transporter cmd/main.go
