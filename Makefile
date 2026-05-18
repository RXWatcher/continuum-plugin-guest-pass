BINARY := continuum-plugin-guest-pass
GO ?= go
PNPM ?= pnpm

.PHONY: build web-deps web-build test clean

build: web-build
	$(GO) build -o $(BINARY) ./cmd/continuum-plugin-guest-pass

web-deps:
	cd web && $(PNPM) install --frozen-lockfile

web-build: web-deps
	cd web && $(PNPM) build

test: test-go test-web

test-go:
	$(GO) test ./...

test-web:
	cd web && $(PNPM) run test --run

clean:
	rm -f $(BINARY)
	rm -rf web/dist web/node_modules
