.PHONY: install run run-rust-client lint clean format

# The Rust CLI is optional: its steps only run when cargo is installed.
CARGO := $(shell command -v cargo 2>/dev/null)
SKIP_RUST = @echo "cargo not found: skipping the Rust CLI"

run:
	@go run CLI

run-rust-client:
ifdef CARGO
	@cargo run
else
	@echo "cargo not found: install Rust to run the Rust CLI, or use make run-client"
	@exit 1
endif

install:
	go build
ifdef CARGO
	cargo build
else
	$(SKIP_RUST)
endif

lint:
	go vet
ifdef CARGO
	cargo check
else
	$(SKIP_RUST)
endif

format:
	gofmt -l -w .
ifdef CARGO
	cargo fmt
else
	$(SKIP_RUST)
endif

clean:
	go clean
ifdef CARGO
	cargo clean
else
	$(SKIP_RUST)
endif
