.PHONY: build test

build:
	go build -o potacast ./cmd/potacast

test: build
	./test.sh --skip-build
