.PHONY: all build clean ui backend run test docker

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-s -w -X github.com/siriusec/siriusec_claw/pkg/version.Version=$(VERSION) -X github.com/siriusec/siriusec_claw/pkg/version.BuildTime=$(BUILD_TIME)"
BINARY := siriusec_claw
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

all: ui backend

# --- Frontend ---

ui:
	@echo "==> Building frontend..."
	cd ui && npm install && npm run build
	@echo "==> Frontend built to src/embed/frontend/"

# --- Backend ---

backend:
	@echo "==> Building $(BINARY) $(VERSION) for $(GOOS)/$(GOARCH)..."
	cd src && CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(LDFLAGS) -o ../bin/$(BINARY) ./cmd/siriusec_claw
	@echo "==> Binary: bin/$(BINARY)"

build: all

# --- Run ---

run: all
	./bin/$(BINARY) gateway --port 3578

run-dev:
	cd src && go run $(LDFLAGS) ./cmd/siriusec_claw gateway --port 3578

# --- Test ---

test:
	cd src && go test ./... -v

test-short:
	cd src && go test ./... -short

# --- Lint ---

lint:
	cd src && go vet ./...

# --- Clean ---

clean:
	rm -rf bin/
	rm -rf src/embed/frontend/assets
	rm -f src/embed/frontend/index.html
	rm -rf ui/node_modules
	@echo "==> Cleaned"

# --- Docker ---

docker:
	docker build -t siriusec-claw:$(VERSION) .

docker-run:
	docker run -it --rm -p 3578:3578 siriusec-claw:$(VERSION)

# --- Cross-compilation ---

build-linux-amd64:
	GOOS=linux GOARCH=amd64 $(MAKE) backend

build-linux-arm64:
	GOOS=linux GOARCH=arm64 $(MAKE) backend

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 $(MAKE) backend

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 $(MAKE) backend

build-all: ui build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64
