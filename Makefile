GO_VERSION := $(shell cat go.mod | grep -E 'go [0-9].[0-9]+' | cut -d ' ' -f 2)
VERSION := $(shell echo $(shell git describe --tags) | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')
RUNNER_BASE_IMAGE_DISTROLESS := gcr.io/distroless/static-debian11

install:
	go mod download

start: 
	go run main.go serve --config config.json

build:
	go build -o bin/interlock-oracle main.go

build-docker:
	@DOCKER_BUILDKIT=1 docker build \
		-t interlock-oracle:latest \
		--secret id=sshKey,src=${HOME}/.ssh/id_rsa \
		--secret id=sshKeyPub,src=${HOME}/.ssh/id_rsa.pub \
		--build-arg GO_VERSION=$(GO_VERSION) \
		--build-arg RUNNER_IMAGE=$(RUNNER_BASE_IMAGE_DISTROLESS) \
		--build-arg GIT_VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(COMMIT) \
		-f Dockerfile .

###############################################################################
###                                Linting                                  ###
###############################################################################

format-tools:
	go install mvdan.cc/gofumpt@v0.4.0
	go install github.com/client9/misspell/cmd/misspell@v0.3.4
	go install github.com/daixiang0/gci@v0.11.2
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sudo sh -s -- -b $(go env GOPATH)/bin v1.57.2

lint: format-tools
	golangci-lint run --tests=false