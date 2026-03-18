.PHONY: build build-all package clean

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS = -ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o potacast ./cmd/potacast

build-all:
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/potacast-linux-amd64 ./cmd/potacast
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/potacast-linux-arm64 ./cmd/potacast
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/potacast-darwin-amd64 ./cmd/potacast
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/potacast-darwin-arm64 ./cmd/potacast

package: build-all
	@mkdir -p release
	@cd dist && for f in potacast-*; do \
		cp $$f potacast && tar -czf ../release/$$f.tar.gz potacast && rm -f potacast; \
	done

clean:
	rm -rf dist release potacast
