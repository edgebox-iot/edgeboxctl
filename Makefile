PROJECT?=github.com/edgebox-iot/edgeboxctl

RELEASE ?= dev
COMMIT := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
BUILD_DIR = bin

build-all:
	make build-arm
	make build-amd64

build-arm:
	GOOS=linux GOARCH=arm make build

build-amd64:
	GOOS=linux GOARCH=amd64 make build

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

install-cloud: build-amd64
	cp ./bin/edgeboxctl-amd64 /usr/local/bin/edgeboxctl
	cp ./edgeboxctl/edgeboxctl.service /lib/systemd/system/edgeboxctl.service
	systemctl daemon-reload
	@echo "Edgeboxctl installed successfully"
	@echo "To start edgeboxctl run: systemctl start edgeboxctl"

install-prod: build-arm
	cp ./bin/edgeboxctl-arm /usr/local/bin/edgeboxctl
	cp ./edgeboxctl/edgeboxctl.service /lib/systemd/system/edgeboxctl.service
	systemctl daemon-reload
	@echo "Edgeboxctl installed successfully"
	@echo "To start edgeboxctl run: systemctl start edgeboxctl"
