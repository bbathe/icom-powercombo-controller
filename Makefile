package := $(shell basename `pwd`)

.PHONY: default get codetest build fmt lint vet

default: fmt codetest

get:
	GOOS=windows GOARCH=amd64 go get -v ./...
	go get github.com/akavel/rsrc
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.31.0

codetest: lint vet

build:
	mkdir -p target
	rm -f target/$(package).exe target/$(package).log
	$(shell go env GOPATH)/bin/rsrc -arch amd64 -manifest $(package).manifest -ico $(package).ico -o cmd/icom-powercombo-controller/$(package).syso
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC="x86_64-w64-mingw32-gcc" go build -v -ldflags "-s -w -H=windowsgui" -o target/$(package).exe github.com/bbathe/icom-powercombo-controller/cmd/icom-powercombo-controller

fmt:
	GOOS=windows GOARCH=amd64 go fmt ./...

lint:
	GOOS=windows GOARCH=amd64 $(shell go env GOPATH)/bin/golangci-lint run --fix

vet:
	GOOS=windows GOARCH=amd64 go vet -all ./...