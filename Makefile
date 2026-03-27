.PHONY: build test lint

build:
	go build -o linkding-cleaner ./cmd/linkding-cleaner/

test:
	go test ./...

lint:
	gofmt -l . && go vet ./...
