BIN=semvereis
TARGET=/usr/local/bin
NEXTMINOR = $(shell go run semvereis.go next minor -d v0.0.0)
NEXTMINORTAG = $(shell go run semvereis.go next minor -d v0.0.0 -v)
NEXTPATCH = $(shell go run semvereis.go next patch -d v0.0.0)
NEXTPATCHTAG = $(shell go run semvereis.go next patch -d v0.0.0 -v)
tar = tar --owner=0 --group=0 -czf $(BIN)-$(1).tar.gz $(BIN) --transform 's|^|$(BIN)-$(1)/|' go.mod go.sum semvereis.go LICENSE Makefile README.md VERSION
tagmsg = @echo "Commit, tag and push: git commit -a ; git tag $(1) ; git push origin $(1)"

.PHONY: all build install clean release patch minor

all: build

build: $(BIN)

$(BIN):
	go build -o $(BIN) -ldflags=-s ./...

install:
	install $(BIN) $(TARGET)/

clean:
	rm -f $(BIN) $(wildcard $(BIN)-*.tar.gz)

release: releaseMinor

releaseMinor: clean
	$(shell echo -n $(NEXTMINOR) > VERSION)
	$(MAKE) build
	$(call tar,$(NEXTMINOR))
	$(call tagmsg,$(NEXTMINORTAG))

releasePatch: clean
	$(shell echo -n $(NEXTPATCH) > VERSION)
	$(MAKE) build
	$(call tar,$(NEXTPATCH))
	$(call tagmsg,$(NEXTPATCHTAG))
