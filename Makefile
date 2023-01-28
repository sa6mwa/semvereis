BIN=semvereis
TARGET=/usr/local/bin
NEXTVER = $(shell ./$(BIN) next minor -d v0.0.0)
NEXTTAG = $(shell ./$(BIN) next minor -d v0.0.0 -v)

.PHONY: all build install clean release

all: build

build: $(BIN)

$(BIN):
	go build -o $(BIN) -ldflags=-s ./...

install:
	install $(BIN) $(TARGET)/

clean:
	rm -f $(BIN) semvereis-*.tar.gz

release: $(BIN)
	$(shell echo -n $(NEXTVER) > VERSION)
	tar --owner=0 --group=0 -czf semvereis-$(NEXTVER).tar.gz $(BIN) --transform 's|^|$(BIN)-$(NEXTVER)/|' go.mod go.sum semvereis.go LICENSE Makefile README.md VERSION
	@echo Tag commit with: git tag $(NEXTTAG)
