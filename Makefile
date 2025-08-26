APP_NAME=adaptive-redis-list

.PHONY: all build test run tidy clean

all: build

## Build the project
build:
	@echo ">> Building $(APP_NAME)"
	go build -o bin/$(APP_NAME) ./src/...

## Run tests
test:
	@echo ">> Running tests"
	go test ./src/test -v

## Run example (if you add a main.go under src/)
run:
	@echo ">> Running example"
	go run ./src/main.go

## Go mod tidy
tidy:
	@echo ">> Running go mod tidy"
	go mod tidy

## Clean build artifacts
clean:
	@echo ">> Cleaning up"
	rm -rf bin

