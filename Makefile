REVISION==$(shell git describe --tags --always)
BUILD=$(shell date +%FT%T%z)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
LDFLAGS=-ldflags "-X main.Revision=${REVISION} -X main.Build=${BUILD} -X main.Branch=${BRANCH}"

HORUS_CLI_DIR=./cmd
HORUS_BIN_DIR=$(HORUS_CLI_DIR)/bin
HORUS_DISPATCHER=horus-dispatcher
HORUS_AGENT=horus-agent
HORUS_QUERY=horus-query

all:
	go build $(LDFLAGS) -o $(HORUS_BIN_DIR) ./...

dispatcher:
	go build $(LDFLAGS) -o $(HORUS_BIN_DIR) $(HORUS_CLI_DIR)/$(HORUS_DISPATCHER)

agent:
	go build $(LDFLAGS) -o $(HORUS_BIN_DIR) $(HORUS_CLI_DIR)/$(HORUS_AGENT)

query:
	go build $(LDFLAGS) -o $(HORUS_BIN_DIR) $(HORUS_CLI_DIR)/$(HORUS_QUERY)

install:
	go install $(LDFLAGS) ./...

test:
	go test -race -short ./...

cov:
	go test -race -cover ./...

clean:
	rm -f $(HORUS_BIN_DIR)/*
	go clean -i -testcache -modcache ./...

.PHONY: all dispatcher agent query install test cov clean
