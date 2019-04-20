# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOINSTALL=$(GOCMD) install
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD)get
BINARY_NAME=main
PROJECT_PATH=github.com/mattsteencpp/go-daily-tracker
EXE_PATH=$(GOPATH)/bin

# all: test build
all: build
build:
	$(GOBUILD) -o $(EXE_PATH)/dt $(PROJECT_PATH)/$(BINARY_NAME)
test: 
	$(GOTEST) -v ./...
clean: 
	$(GOCLEAN)
	rm -f $(EXE_PATH)/$(BINARY_NAME)
format:
	gofmt -w .
run:
	$(BINARY_NAME)

