.PHONY: install run run-rust-client lint clean format

run:
	@go run CLI

run-rust-client:
	@cargo run

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
