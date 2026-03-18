.PHONY: build build-all package test

PLATFORMS := linux-amd64 linux-arm64 darwin-amd64 darwin-arm64
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

build:
	go build -o potacast ./cmd/potacast

build-all:
	@mkdir -p dist
	@for p in $(PLATFORMS); do \
		GOOS=$${p%-*} GOARCH=$${p#*-} go build -o dist/potacast-$$p ./cmd/potacast; \
	done

package: build-all
	@mkdir -p release
	@for p in $(PLATFORMS); do \
		mkdir -p release/$$p && cp dist/potacast-$$p release/$$p/potacast && \
		tar -czf release/potacast-$$p.tar.gz -C release/$$p potacast && rm -rf release/$$p; \
	done

test: build
	./test.sh --skip-build
