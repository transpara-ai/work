.PHONY: build test cover vet verify

build:
	go build ./cmd/...

test:
	go test ./...

cover:
	go test -cover ./...

vet:
	go vet ./...

verify: build test vet
