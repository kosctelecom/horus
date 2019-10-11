REVISION=`git describe --tags --always`
BUILD=`date +%FT%T%z`
BRANCH=`git rev-parse --abbrev-ref HEAD`
LDFLAGS=-ldflags "-X main.Revision=${REVISION} -X main.Build=${BUILD} -X main.Branch=${BRANCH}"

HORUS_CLI_DIR=./cmd
HORUS_BIN_DIR=$(HORUS_CLI_DIR)/bin
HORUS_DISPATCHER=horus-dispatcher
HORUS_AGENT=horus-agent
HORUS_QUERY=horus-query

all: dispatcher agent query

dispatcher:
	go build $(LDFLAGS) -o $(HORUS_BIN_DIR) $(HORUS_CLI_DIR)/$(HORUS_DISPATCHER)

agent:
	go build $(LDFLAGS) -o $(HORUS_BIN_DIR) $(HORUS_CLI_DIR)/$(HORUS_AGENT)

query:
	go build $(LDFLAGS) -o $(HORUS_BIN_DIR) $(HORUS_CLI_DIR)/$(HORUS_QUERY)

test:
	go test -race -short ./dispatcher ./agent ./model

clean:
	@rm -f $(HORUS_BIN_DIR)/*

.PHONY: all dispatcher agent query test clean
