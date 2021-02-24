BUILD_VERSION ?= $(shell git describe --always --abbrev=40 --dirty)

PREFIX?=/usr/local
BINDIR?=$(PREFIX)/bin
SHAREDIR?=$(PREFIX)/share/smithy

LDFLAGS="-X github.com/honza/smithy/cmd.SmithyVersion=${BUILD_VERSION}"
MODCACHE := $(shell go env GOMODCACHE)

export CGO_ENABLED=0

all: smithy smithy.yml

smithy: bin/statik
	bin/statik -src=include -dest=pkg -f -m
	go build -ldflags $(LDFLAGS) -o smithy main.go

smithy.yml:
	./smithy generate > smithy.yml

install: all
	mkdir -m755 -p $(DESTDIR)$(BINDIR) $(DESTDIR)$(SHAREDIR)
	install -m755 smithy $(DESTDIR)$(BINDIR)/smithy
	install -m644 smithy.yml $(DESTDIR)$(SHAREDIR)/smithy.yml

uninstall: all
	rm -r $(DESTDIR)$(BINDIR)/smithy
	rm -fr $(DESTDIR)$(SHAREDIR)

bin/statik:
	mkdir -p bin
	go mod download
	go build -o bin/statik $(MODCACHE)/github.com/rakyll/statik@v0.1.7/statik.go

gofmt:
	go fmt ./pkg/... ./cmd/...

clean:
	rm smithy smithy.yml
	rm -rf pkg/statik

.PHONY:
	smithy smithy.yml clean
