.PHONY: all

all: check build

check:
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.42.1
	golangci-lint run

build: checksum create-wrapper mv-wrapper rm-wrapper transporter

checksum:
	go build -o cmd/checksum/checksum cmd/checksum/main.go

create-wrapper:
	go build -o cmd/create_wrapper/create-wrapper cmd/create_wrapper/main.go

mv-wrapper:
	go build -o cmd/mv_wrapper/mv-wrapper cmd/mv_wrapper/main.go

rm-wrapper:
	go build -o cmd/rm_wrapper/rm-wrapper cmd/rm_wrapper/main.go

transporter:
	go build -o cmd/transporter/transporter cmd/transporter/main.go