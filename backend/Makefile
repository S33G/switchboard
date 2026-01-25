.PHONY: help run test fmt tidy swagger build clean init-config

ACCESS_LOGS ?= 1

help:
	@printf "Targets:\n"
	@printf "  run      - run the backend\n"
	@printf "  init-config - write a YAML config file (CONFIG_OUT=path)\n"
	@printf "  test     - run tests\n"
	@printf "  fmt      - gofmt all Go files\n"
	@printf "  tidy     - go mod tidy\n"
	@printf "  swagger  - generate swagger files\n"
	@printf "  build    - build backend binary\n"
	@printf "  clean    - remove build artifacts\n"

run:
	API_PORT=6969 CONFIG_PATH=./config.yaml ACCESS_LOGS=$(ACCESS_LOGS) go run .

CONFIG_OUT ?= config.yaml
init-config:
	go run . -init-config $(CONFIG_OUT)

test:
	go test ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy

swagger:
	swag init -g swagger.go -o docs --parseDependency --parseInternal

build:
	go build -o bin/switchboard-backend .

clean:
	rm -rf bin

deps:
	go mod download
