.PHONY: run build test test-cover lint lint-fix race vet

run:
	go run -buildvcs=false ./src

build:
	mkdir -p bin
	go build -buildvcs=false -o bin/lab3 ./src

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
