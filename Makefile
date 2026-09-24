BINARY_NAME := tofu-key-provider-gopass
BUILD_VERSION != git describe --tags --always --dirty
BUILD_DATE != date -u +%Y-%m-%dT%H:%M:%SZ

.PHONY: all build lint test test-race clean

all: build

build: $(BINARY_NAME)

$(BINARY_NAME): main.go go.mod go.sum
	go build \
		-trimpath \
		-ldflags="-s -w -X main.version=$(BUILD_VERSION) -X main.date=$(BUILD_DATE)" \
		-o $@ $<

lint:
	go vet ./...
	golangci-lint run

test:
	go test ./...

test-race:
	go test -race ./...

clean:
	rm -f $(BINARY_NAME)
