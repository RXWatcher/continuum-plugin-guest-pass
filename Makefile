BINARY := continuum-plugin-guest-pass
GO ?= go
PNPM ?= pnpm

.PHONY: build web-deps web-build test clean

build: web-build
	$(GO) build -o $(BINARY) ./cmd/continuum-plugin-guest-pass

web-deps:
	cd web && $(PNPM) install

web-build: web-deps
	cd web && $(PNPM) build

test:
	$(GO) test ./...

clean:
	rm -f $(BINARY)
	rm -rf web/dist web/node_modules
