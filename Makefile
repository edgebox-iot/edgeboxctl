PROJECT?=github.com/edgebox-iot/edgeboxctl

RELEASE ?= dev
COMMIT := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
BUILD_DIR = bin

build-all:
	GOOS=linux GOARCH=amd64 make build
	GOOS=linux GOARCH=arm make build

build-prod:
	GOOS=linux GOARCH=arm RELEASE=prod make build

build-cloud:
	GOOS=linux GOARCH=amd64 RELEASE=cloud make build

build:
	@echo "Building ${GOOS}-${GOARCH}"
	GOOS=${GOOS} GOARCH=${GOARCH} go build \
		-trimpath -ldflags "-s -w -X ${PROJECT}/internal/diagnostics.Version=${RELEASE} \
		-X ${PROJECT}/internal/diagnostics.Commit=${COMMIT} \
		-X ${PROJECT}/internal/diagnostics.BuildDate=${BUILD_DATE}" \
		-o bin/edgeboxctl-${GOOS}-${GOARCH} ${PROJECT}/cmd/edgeboxctl
	cp ./bin/edgeboxctl-${GOOS}-${GOARCH} ./bin/edgeboxctl

clean:
	rm -rf ${BUILD_DIR}
	go clean

test:
	go test -tags=unit -timeout=600s -v ./...

test-with-coverage:
	go test -tags=unit -timeout=600s -v ./... -coverprofile=coverage.out
