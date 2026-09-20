.PHONY: install run lint clean format

run:
	@go run CLI

install:
	go build

lint:
	go vet
	cargo check

format:
	gofmt -l -w .
	cargo fmt

clean:
	go clean