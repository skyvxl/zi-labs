.PHONY: run build test race vet

run:
	go run ./src

build:
	mkdir -p bin
	go build -o bin/lab1 ./src

test:
	go test ./...

test-cover:
	go test -cover ./...

lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

race:
	go test -race ./...

vet:
	go vet ./...
