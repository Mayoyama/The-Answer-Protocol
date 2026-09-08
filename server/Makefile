.PHONY: install run lint clean format

run:
	go run server

install:
	go mod download
	go build

lint:
	go vet

format:
	gofmt -l -w .

clean:
	go clean