.PHONY: all build clean install release

BINARY_NAME=ttsalert
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}"

all: build

build:
	@echo "Building ${BINARY_NAME}..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME} ./cmd

build-arm64:
	@echo "Building ${BINARY_NAME} for ARM64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY_NAME}-arm64 ./cmd

build-windows:
	@echo "Building ${BINARY_NAME} for Windows..."
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}.exe ./cmd

clean:
	@echo "Cleaning..."
	rm -f ${BINARY_NAME} ${BINARY_NAME}-arm64 ${BINARY_NAME}.exe
	rm -rf dist/

install: build
	@echo "Installing..."
	install -m 755 ${BINARY_NAME} /usr/local/bin/
	install -d /etc/ttsalert
	install -m 640 configs/config.example.yaml /etc/ttsalert/config.yaml
	install -d -m 755 /var/lib/ttsalert/audio
	install -m 644 ttsalert.service /etc/systemd/system/
	systemctl daemon-reload
	@echo "Installation complete"
	@echo "Edit /etc/ttsalert/config.yaml and run: systemctl enable --now ttsalert"

release: clean
	@echo "Creating release builds..."
	mkdir -p dist
	
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}_linux_amd64 ./cmd
	tar -czf dist/${BINARY_NAME}_linux_amd64.tar.gz -C dist ${BINARY_NAME}_linux_amd64
	
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o dist/${BINARY_NAME}_linux_arm64 ./cmd
	tar -czf dist/${BINARY_NAME}_linux_arm64.tar.gz -C dist ${BINARY_NAME}_linux_arm64
	
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}_windows_amd64.exe ./cmd
	zip -j dist/${BINARY_NAME}_windows_amd64.zip dist/${BINARY_NAME}_windows_amd64.exe
	
	@echo "Release builds created in dist/"

test:
	@echo "Running tests..."
	go test -v ./...

lint:
	@echo "Running linter..."
	golangci-lint run

run:
	@echo "Running ${BINARY_NAME}..."
	go run ./cmd

help:
	@echo "Available targets:"
	@echo "  build        - Build for Linux amd64"
	@echo "  build-arm64  - Build for Linux arm64"
	@echo "  build-windows - Build for Windows"
	@echo "  clean        - Remove build artifacts"
	@echo "  install      - Install to system"
	@echo "  release      - Create release packages"
	@echo "  test         - Run tests"
	@echo "  lint         - Run linter"
	@echo "  run          - Run locally"
	@echo "  help         - Show this help"
