.PHONY: build test generate lint

build:
	go build -o bin/sobrevoo ./cmd/sobrevoo

test:
	go test ./... -cover

generate:
	go generate ./...

lint:
	go vet ./...
