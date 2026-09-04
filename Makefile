package := $(shell basename `pwd`)

.PHONY: default get codetest build fmt lint vet test

default: fmt codetest

get:
	GOOS=windows GOARCH=amd64 go mod download
	go install github.com/akavel/rsrc@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.9.0

codetest: lint vet test

# Native host tests (not GOOS=windows): Windows test binaries cannot run on Linux CI.
# Skip ui/cmd — they require Walk/Win32.
test:
	go test ./config ./controller ./data ./device/... ./status ./util

build:
	mkdir -p target
	rm -f target/$(package).exe target/$(package).log
	$(shell go env GOPATH)/bin/rsrc -arch amd64 -manifest $(package).manifest -ico $(package).ico -o cmd/icom-powercombo-controller/$(package).syso
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC="x86_64-w64-mingw32-gcc" go build -v -ldflags "-s -w -H=windowsgui" -o target/$(package).exe github.com/bbathe/icom-powercombo-controller/cmd/icom-powercombo-controller

fmt:
	GOOS=windows GOARCH=amd64 go fmt ./...

lint:
	GOOS=windows GOARCH=amd64 $(shell go env GOPATH)/bin/golangci-lint run --timeout 5m

vet:
	GOOS=windows GOARCH=amd64 go vet -all ./...
