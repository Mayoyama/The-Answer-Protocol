.PHONY: install run run-go-client lint clean format

run:
	@cargo run

run-go-client:
	@go run CLI

install:
	go build
	cargo build

lint:
	go vet
	cargo check

format:
	gofmt -l -w .
	cargo fmt

clean:
	go clean
	cargo clean