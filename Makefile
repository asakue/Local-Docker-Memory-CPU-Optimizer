.PHONY: all build test clean run demo summary

BINARY_NAME=dockertrim

all: test build

build:
	go build -ldflags="-s -w" -o $(BINARY_NAME) .

test:
	go test -v ./...

run:
	go run .

demo:
	go run . --demo

summary:
	go run . --summary

clean:
	go clean
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe dockertrim.log

