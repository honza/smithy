BUILD_VERSION ?= $(shell git describe --always --abbrev=40 --dirty)

LDFLAGS="-X github.com/honza/smithy/cmd.SmithyVersion=${BUILD_VERSION}"

all:
	statik -src=include -dest=pkg -f -m
	CGO_ENABLED=0 go build -ldflags $(LDFLAGS) -o smithy main.go

gofmt:
	go fmt ./pkg/... ./cmd/...
