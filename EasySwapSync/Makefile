SHELL := /bin/bash

build:
	@echo "build start..."
	@mkdir -p build
	go build main.go
	@mv main build/sync_service
	@echo "build done!"

run: build
	@echo "run sync service"
	./build/sync_service daemon -c "./config/config_import.toml"

build_linux:
	@echo "build linux amd64 start..."
	@mkdir -p build
	GOOS=linux GOARCH=amd64 go build main.go
	@mv main build/sync_service_linux
	@echo "build linux amd64 done!"
.PHONY: build