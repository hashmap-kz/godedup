APP_NAME 	 := godedup
OUTPUT   	 := $(APP_NAME)
COV_REPORT 	 := coverage.txt
INSTALL_DIR  := /usr/local/bin

ifeq ($(OS),Windows_NT)
	OUTPUT := $(APP_NAME).exe
endif

.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/$(OUTPUT) main.go

.PHONY: lint
lint:
	golangci-lint run --output.tab.path=stdout

.PHONY: install
install: build
	@echo "Installing bin/$(OUTPUT) to $(INSTALL_DIR)..."
	@sudo chmod +x bin/$(OUTPUT) && sudo cp bin/$(OUTPUT) $(INSTALL_DIR)

.PHONY: test
test:
	go test -v -race -cover ./...

.PHONY: clean
clean:
	@rm -rf bin/ dist/ *.log

.PHONY: snapshot
snapshot:
	GORELEASER_FORCE_TOKEN=github goreleaser release --skip sign --skip publish --snapshot --clean

.PHONY: help
help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z0-9_.-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2}'
