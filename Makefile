.PHONY: build test vet cover check

build:
	go build -o bin/cardmaker ./cmd/cardmaker

test:
	go test ./...

vet:
	go vet ./...

cover:
	go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1

check: vet test
