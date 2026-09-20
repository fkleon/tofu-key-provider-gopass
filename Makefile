BINARY_NAME := tofu-key-provider-gopass

BUILD_VERSION != git describe --tags --always --dirty
BUILD_DATE != date -u +%Y-%m-%dT%H:%M:%SZ

$(BINARY_NAME): main.go
	go build \
		-trimpath \
		-ldflags="-s -w -X main.version=$(BUILD_VERSION) -X main.date=$(BUILD_DATE)" \
		-o $@ $<

.PHONY: lint
lint:
	go vet ./...
	golangci-lint run

.PHONY: test
test:
	go test ./...
