GOPATH := $(shell go env GOPATH)
PATH   := $(HOME)/bin:$(GOPATH)/bin:$(PATH)

.PHONY: proto build run tidy

proto:
	cd .. && buf dep update && buf generate

build:
	go build -o server ./cmd/server

run:
	go run ./cmd/server

tidy:
	go mod tidy
