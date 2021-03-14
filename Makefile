BUILD_VERSION ?= $(shell git describe --always --abbrev=40 --dirty)

SCDOC = scdoc
PREFIX?=/usr/local
BINDIR?=$(PREFIX)/bin
SHAREDIR?=$(PREFIX)/share/smithy
MANDIR?=$(PREFIX)/share/man

LDFLAGS="-X github.com/honza/smithy/cmd.SmithyVersion=${BUILD_VERSION}"
MODCACHE := $(shell go env GOMODCACHE)

export CGO_ENABLED=0

all: smithy smithy.yml

smithy: bin/statik include/*.html
	bin/statik -src=include -dest=pkg -f -m
	go build -ldflags $(LDFLAGS) -o smithy main.go

smithy.yml:
	./smithy generate > smithy.yml

docs:
	$(SCDOC) < docs/smithy.1.scd > smithy.1
	$(SCDOC) < docs/smithy.yml.5.scd > smithy.yml.5

install: all
	mkdir -m755 -p $(DESTDIR)$(BINDIR) $(DESTDIR)$(SHAREDIR)
	cp -f smithy $(DESTDIR)$(BINDIR)/smithy
	cp -f smithy.yml $(DESTDIR)$(SHAREDIR)/smithy.yml
	cp -f smithy.1 $(DESTDIR)$(MANDIR)/man1/smithy.1 2>/dev/null || true
	cp -f smithy.yml.5 $(DESTDIR)$(MANDIR)/man5/smithy.yml.5 2>/dev/null || true

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
	rm -rf smithy smithy.yml pkg/statik smithy.1 smithy.yml.5

.PHONY:
	smithy smithy.yml clean
