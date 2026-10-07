.PHONY: install run lint clean format

run:
	go run GUI

install:
	go mod download
	go build

lint:
	go vet

format:
	gofmt -l -w .

clean:
	go clean