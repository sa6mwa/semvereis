BIN=semvereis
TARGET=/usr/local/bin
VF = VERSION
NEXTMINOR = go run semvereis.go next minor -nd v0.0.0
NEXTMINORTAG = go run semvereis.go next minor -nvd v0.0.0
NEXTPATCH = go run semvereis.go next patch -nd v0.0.0
NEXTPATCHTAG = go run semvereis.go next patch -nvd v0.0.0
tar = tar --owner=0 --group=0 -czf $(BIN)-$(1).tar.gz $(BIN) --transform 's|^|$(BIN)-$(1)/|' go.mod go.sum semvereis.go LICENSE Makefile README.md VERSION
tagmsg = @echo "Commit, tag and push: git commit -a ; git tag $(1) ; git push origin $(1)"

.PHONY: all build install clean release releaseMinor releasePatch

all: build

build: $(BIN)

$(BIN):
	go build -o $(BIN) -ldflags=-s ./...

install:
	install $(BIN) $(TARGET)/

clean:
	rm -f $(BIN) $(wildcard $(BIN)-*.tar.gz)

$(VF):
	git describe --tags --abbrev=0 | sed 's/^v//' > $(VF)

release: releaseMinor

releaseMinor: clean
	$(NEXTMINOR) -so $(VF)
	$(MAKE) build
	$(call tar,$(shell $(NEXTMINOR)))
	$(call tagmsg,$(shell $(NEXTMINORTAG)))

releasePatch: clean
	$(NEXTPATCH) -so $(VF)
	$(MAKE) build
	$(call tar,$(shell $(NEXTPATCH)))
	$(call tagmsg,$(shell $(NEXTPATCHTAG)))
