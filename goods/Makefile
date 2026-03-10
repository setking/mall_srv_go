# Makefile
.PHONY: build-linux

build-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/myapp_linux

build-windows:
	GOOS=windows GOARCH=amd64 go build -o bin/myapp.exe

build-all: build-linux build-windows