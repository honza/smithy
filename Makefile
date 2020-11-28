BUILD_VERSION ?= $(shell git describe --always --abbrev=40 --dirty)

LDFLAGS="-X github.com/honza/smithy/cmd.SmithyVersion=${BUILD_VERSION}"
MODCACHE := $(shell go env GOMODCACHE)

export CGO_ENABLED=0

all: bin/statik
	bin/statik -src=include -dest=pkg -f -m
	go build -ldflags $(LDFLAGS) -o smithy main.go

bin/statik:
	mkdir -p bin
	go mod download
	go build -o bin/statik $(MODCACHE)/github.com/rakyll/statik@v0.1.7/statik.go

gofmt:
	go fmt ./pkg/... ./cmd/...

clean:
	rm smithy
	rm -rf pkg/statik
