PROJECT?=github.com/edgebox-iot/sysctl

RELEASE ?= dev
COMMIT := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
BUILD_DIR = bin

DBHOST = 127.0.0.1:3306
DBNAME = docker
DBUSER = root
DBPASS = tiger

build-all:
	GOOS=linux GOARCH=amd64 make build
	GOOS=linux GOARCH=arm make build

build:
	@echo "Building ${GOOS}-${GOARCH}"
	GOOS=${GOOS} GOARCH=${GOARCH} go build \
		-trimpath -ldflags "-s -w -X ${PROJECT}/internal/diagnostics.Version=${RELEASE} \
		-X ${PROJECT}/internal/diagnostics.Commit=${COMMIT} \
		-X ${PROJECT}/internal/diagnostics.BuildDate=${BUILD_DATE} \
		-X ${PROJECT}/internal/database.Version=${RELEASE} \
		-X ${PROJECT}/internal/database.Commit=${COMMIT} \
		-X ${PROJECT}/internal/database.BuildDate=${BUILD_DATE} \
		-X ${PROJECT}/internal/database.Dbhost=${DBHOST} \
		-X ${PROJECT}/internal/database.Dbname=${DBNAME} \
		-X ${PROJECT}/internal/database.Dbuser=${DBUSER} \
		-X ${PROJECT}/internal/database.Dbpass=${DBPASS}" \
		-o bin/sysctl-${GOOS}-${GOARCH} ${PROJECT}/cmd/sysctl
	cp ./bin/sysctl-${GOOS}-${GOARCH} ./bin/sysctl

clean:
	rm -rf ${BUILD_DIR}
	go clean
