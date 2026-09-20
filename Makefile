export CGO_ENABLED=0
BINARY_NAME := tofu-key-provider-gopass

$(BINARY_NAME): main.go
	go build -trimpath -ldflags="-s -w" -o $@ $<

.PHONY: lint
lint:
	go vet ./...
	golangci-lint run

.PHONY: test
test:
	go test ./...
