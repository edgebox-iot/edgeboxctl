.DEFAULT_GOAL := build

PROJECT ?= github.com/edgebox-iot/edgeboxctl
RELEASE ?= dev
COMMIT := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
BUILD_DIR = bin
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)


build-all:
	@echo "\n🏗️ Building all architectures for ${RELEASE} mode"
	@echo "🟡 This will build all supported architectures and release combinations. It can take a while...\n"

	GOOS=linux GOARCH=amd64 make build
	GOOS=linux GOARCH=arm make build

	@echo "\n🟢 All builds completed and available at ./bin/ \n"

build-prod:
	GOOS=linux GOARCH=arm RELEASE=prod make build

build-cloud:
	GOOS=linux GOARCH=amd64 RELEASE=cloud make build

build-arm64:
	GOOS=linux GOARCH=arm64 RELEASE=prod make build

build-armhf:
	GOOS=linux GOARCH=arm RELEASE=prod make build

build-amd64:
	GOOS=linux GOARCH=amd64 RELEASE=prod make build


build:
	@echo "\n🏗️ Building edgeboxctl (${RELEASE} release) on ${GOOS} (${GOARCH})"
	@echo "📦 Binary will be saved in ./${BUILD_DIR}/edgeboxctl-${GOOS}-${GOARCH}\n"

	GOOS=${GOOS} GOARCH=${GOARCH} go build \
		-trimpath -ldflags "-s -w -X ${PROJECT}/internal/diagnostics.Version=${RELEASE} \
		-X ${PROJECT}/internal/diagnostics.Commit=${COMMIT} \
		-X ${PROJECT}/internal/diagnostics.BuildDate=${BUILD_DATE}" \
		-o bin/edgeboxctl-${GOOS}-${GOARCH} ${PROJECT}/cmd/edgeboxctl

	@echo "\n🟢 Build task completed\n"

clean:
	@echo "🧹 Cleaning build directory and go cache\n"

	rm -rf ${BUILD_DIR}
	go clean

	@echo "\n🟢 Clean task completed\n"

test:
	go test -tags=unit -timeout=600s -v ./...

test-with-coverage:
	go test -tags=unit -timeout=600s -v ./... -coverprofile=coverage.out

run:
	@echo "\n🚀 Running edgeboxctl\n"
	./bin/edgeboxctl-${GOOS}-${GOARCH}

install:
	@echo "📦 Installing edgeboxctl service (${RELEASE}) for ${GOOS} (${GOARCH})\n"
	
	@echo "�🚧 Stopping edgeboxctl service if it is running"
	sudo systemctl stop edgeboxctl || true

	@echo "\n🗑️ Removing old edgeboxctl binary and service"
	sudo rm -rf /usr/local/bin/edgeboxctl /usr/local/sbin/edgeboctl /lib/systemd/system/edgeboxctl.service
	
	@echo "\n🚚 Copying edgeboxctl binary to /usr/local/bin"
	sudo cp ./bin/edgeboxctl-${GOOS}-${GOARCH} /usr/local/bin/edgeboxctl
	sudo cp ./bin/edgeboxctl-${GOOS}-${GOARCH} /usr/local/sbin/edgeboxctl

	@echo "\n🚚 Copying edgeboxctl service to /lib/systemd/system"
	sudo cp ./edgeboxctl.service /lib/systemd/system/edgeboxctl.service
	sudo systemctl daemon-reload

	@echo "\n 🚀 To start edgeboxctl run: make start"
	@echo "🟢 Edgeboxctl installed successfully\n"

install-prod: build-prod install
install-cloud: build-cloud install
install-arm64: build-arm64 install
install-armhf: build-armhf install
install-amd64: build-amd64 install

start:
	@echo "\n 🚀 Starting edgeboxctl service\n"
	systemctl start edgeboxctl
	@echo "\n 🟢 Edgebox service started\n"

stop:
	@echo "\n✋ Stopping edgeboxctl service\n"
	systemctl stop edgeboxctl
	@echo "\n 🟢 Edgebox service stopped\n"

restart:
	@echo "\n💫 Restarting edgeboxctl service\n"
	systemctl restart edgeboxctl
	@echo "\n 🟢 Edgebox service restarted\n"

status:
	@echo "\nℹ️ edgeboxctl Service Info:\n"
	systemctl status edgeboxctl

log:
	@echo "\n📰 edgeboxctl service logs:\n"
	journalctl -fu edgeboxctl
