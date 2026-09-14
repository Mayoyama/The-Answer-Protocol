.PHONY: install run lint clean format

run:
	@go run CLI

install:
	go build

lint:
	go vet

format:
	gofmt -l -w .

clean:
	go clean