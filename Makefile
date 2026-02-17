.PHONY: build install clean release-dry

BINARY=commitai
VERSION=$(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X github.com/kaiqui/commitai/cmd.Version=$(VERSION) -X github.com/kaiqui/commitai/cmd.Commit=$(COMMIT) -s -w"

build:
	go build $(LDFLAGS) -o $(BINARY) .

install: build
	sudo mv $(BINARY) /usr/local/bin/$(BINARY)
	@echo "✅ $(BINARY) installed to /usr/local/bin/"

build-all:
	@mkdir -p dist
	GOOS=linux  GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY)_linux_amd64 .
	GOOS=linux  GOARCH=arm64  go build $(LDFLAGS) -o dist/$(BINARY)_linux_arm64 .
	GOOS=darwin GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY)_darwin_amd64 .
	GOOS=darwin GOARCH=arm64  go build $(LDFLAGS) -o dist/$(BINARY)_darwin_arm64 .
	@echo "✅ Built for all platforms in dist/"

clean:
	rm -f $(BINARY)
	rm -rf dist/

release-dry: build
	./$(BINARY) release --auto --dry-run

test:
	go test ./...

vet:
	go vet ./...
