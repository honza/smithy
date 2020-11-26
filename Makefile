BUILD_VERSION ?= $(shell git describe --always --abbrev=40 --dirty)

LDFLAGS="-X github.com/honza/smithy/cmd.SmithyVersion=${BUILD_VERSION}"

all: bin/statik
	bin/statik -src=include -dest=pkg -f -m
	CGO_ENABLED=0 go build -ldflags $(LDFLAGS) -o smithy main.go

bin/statik:
	mkdir -p bin
	go mod download
	go build -o bin/statik $(GOPATH)/src/github.com/rakyll/statik/statik.go

gofmt:
	go fmt ./pkg/... ./cmd/...
