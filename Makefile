VERSION = $(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
OSARCH=$(shell go env GOHOSTOS)-$(shell go env GOHOSTARCH)

NANOCMD=\
	nanocmd-darwin-amd64 \
	nanocmd-darwin-arm64 \
	nanocmd-linux-amd64

my: nanocmd-$(OSARCH)

docker: nanocmd-linux-amd64

$(NANOCMD): cmd/nanocmd
	GOOS=$(word 2,$(subst -, ,$@)) GOARCH=$(word 3,$(subst -, ,$(subst .exe,,$@))) go build $(LDFLAGS) -o $@ ./$<

nanocmd-%-$(VERSION).zip: nanocmd-%.exe
	rm -rf $(subst .zip,,$@)
	mkdir $(subst .zip,,$@)
	ln $^ $(subst .zip,,$@)
	zip -r $@ $(subst .zip,,$@)
	rm -rf $(subst .zip,,$@)

nanocmd-%-$(VERSION).zip: nanocmd-%
	rm -rf $(subst .zip,,$@)
	mkdir $(subst .zip,,$@)
	ln $^ $(subst .zip,,$@)
	zip -r $@ $(subst .zip,,$@)
	rm -rf $(subst .zip,,$@)

clean:
	rm -rf nanocmd-*

release: \
	nanocmd-darwin-amd64-$(VERSION).zip \
	nanocmd-darwin-arm64-$(VERSION).zip \
	nanocmd-linux-amd64-$(VERSION).zip

test:
	go test -v -cover -race ./...

.PHONY: my docker $(NANOCMD) clean release test
