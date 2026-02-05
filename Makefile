.PHONY: help build-stub build-all dev build test test-coverage clean

help:
	@echo "VRCVideoCacher Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build-stub      Build yt-dlp stub executable"
	@echo "  dev             Run development server"
	@echo "  build           Build production executable"
	@echo "  build-all       Build stub + production"
	@echo "  test            Run all tests"
	@echo "  test-coverage   Run tests with coverage report"
	@echo "  clean           Clean build artifacts"

build-stub:
	@echo "Building yt-dlp-stub..."
	@GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o resources/ytdlp-stub.exe ./cmd/ytdlp-stub

dev: build-stub
	@echo "Starting development server..."
	@"/c/Users/Yuzuki Kana/go/bin/wails.exe" dev

build: build-stub
	@echo "Building production executable..."
	@"/c/Users/Yuzuki Kana/go/bin/wails.exe" build -platform windows/amd64 -ldflags "-s -w"

build-all: build-stub build

test:
	@echo "Running tests..."
	@go test ./... -v

test-coverage:
	@echo "Running tests with coverage..."
	@go test -cover ./...
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf build/bin
	@rm -f resources/ytdlp-stub.exe
	@rm -f coverage.out
	@echo "Clean complete."
