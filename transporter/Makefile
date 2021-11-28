.PHONY: all

all: check build

check:
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.42.1
	golangci-lint run

build: checksum create-wrapper mv-wrapper rm-wrapper stat-wrapper copy copylist

checksum:
	go build -o cmd/checksum/checksum cmd/checksum/main.go

create-wrapper:
	go build -o cmd/create_wrapper/create-wrapper cmd/create_wrapper/main.go

mv-wrapper:
	go build -o cmd/mv_wrapper/mv-wrapper cmd/mv_wrapper/main.go

rm-wrapper:
	go build -o cmd/rm_wrapper/rm-wrapper cmd/rm_wrapper/main.go

stat-wrapper:
	go build -o cmd/stat_wrapper/stat-wrapper cmd/stat_wrapper/main.go

copy:
	go build -o cmd/copy/copy cmd/copy/main.go

copylist:
	go build -o cmd/copylist/copylist cmd/copylist/main.go


clean: checksum-clean create-wrapper-clean mv-wrapper-clean rm-wrapper-clean stat-wrapper-clean copy-clean copylist-clean

checksum-clean:
	rm -f cmd/checksum/checksum

create-wrapper-clean:
	rm -f cmd/create_wrapper/create-wrapper

mv-wrapper-clean:
	rm -f cmd/mv_wrapper/mv-wrapper

rm-wrapper-clean:
	rm -f cmd/rm_wrapper/rm-wrapper

stat-wrapper-clean:
	rm -f cmd/stat_wrapper/stat-wrapper

copy-clean:
	rm -f cmd/copy/copy

copylist-clean:
	rm -f cmd/copylist/copylist

docker:
	docker build -t rsync-wrapper:latest --rm . --no-cache