GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTOOL=$(GOCMD) tool
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=tinker
BINARY_UNIX=$(BINARY_NAME)_unix
BIN_DIR=./bin

all: test build
build: 
	$(GOBUILD) -o $(BIN_DIR)/$(BINARY_NAME) -v
test: 
	$(GOTEST) ./...
coverage:
	$(GOTEST) ./... -coverprofile=coverage.out
	$(GOTOOL) cover -html=coverage.out
benchmark:
	$(GOTEST) ./... -bench=. -benchmem
clean: 
	$(GOCLEAN)
	rm -f $(BIN_DIR)/$(BINARY_NAME)
	rm -f $(BIN_DIR)/$(BINARY_UNIX)
# run:
# 	$(GORUN) 

# Cross compilation
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BIN_DIR)/$(BINARY_UNIX) -v
# run/new:
# 	go run ./main.go
# run/latest:
# 	go run ./main.go -n=false
# run/gemini:
# 	go run ./main.go --provider=google
# debug/cli:
# 	go run ./main.go --tui=false
# serve:
# 	go run ./main.go serve
# list/models:
# 	go run ./main.go list
# list/conversations:
# 	go run ./main.go conversation -l
# build:
# 	$(eval VERSION := $(shell cat VERSION))
# 	go build -ldflags="-s -X 'github.com/honganh1206/tinker/cmd.Version=$(VERSION)'" -o bin/tinker main.go

