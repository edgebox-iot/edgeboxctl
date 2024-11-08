.DEFAULT_GOAL := build

PROJECT ?= github.com/edgebox-iot/edgeboxctl
RELEASE ?= dev
COMMIT := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
BUILD_DIR = bin
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)


build-all:
	GOOS=linux GOARCH=amd64 make build
	GOOS=linux GOARCH=arm make build

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
	@echo "Building ${GOOS}-${GOARCH}"
	GOOS=${GOOS} GOARCH=${GOARCH} go build \
		-trimpath -ldflags "-s -w -X ${PROJECT}/internal/diagnostics.Version=${RELEASE} \
		-X ${PROJECT}/internal/diagnostics.Commit=${COMMIT} \
		-X ${PROJECT}/internal/diagnostics.BuildDate=${BUILD_DATE}" \
		-o bin/edgeboxctl-${GOOS}-${GOARCH} ${PROJECT}/cmd/edgeboxctl

clean:
	rm -rf ${BUILD_DIR}
	go clean

test:
	go test -tags=unit -timeout=600s -v ./...

test-with-coverage:
	go test -tags=unit -timeout=600s -v ./... -coverprofile=coverage.out

install:
	sudo systemctl stop edgeboxctl || true
	sudo rm -rf /usr/local/bin/edgeboxctl /usr/local/sbin/edgeboctl /lib/systemd/system/edgeboxctl.service
	sudo cp ./bin/edgeboxctl-${GOOS}-${GOARCH} /usr/local/bin/edgeboxctl
	sudo cp ./bin/edgeboxctl-${GOOS}-${GOARCH} /usr/local/sbin/edgeboxctl
	sudo cp ./edgeboxctl.service /lib/systemd/system/edgeboxctl.service
	sudo systemctl daemon-reload
	@echo "Edgeboxctl installed successfully"
	@echo "To start edgeboxctl run: systemctl start edgeboxctl"

install-prod: build-prod install
install-cloud: build-cloud install
install-arm64: build-arm64 install
install-armhf: build-armhf install
install-amd64: build-amd64 install

start:
	systemctl start edgeboxctl

stop:
	systemctl stop edgeboxctl

status:
	systemctl status edgeboxctl

log: start
	journalctl -fu edgeboxctl
