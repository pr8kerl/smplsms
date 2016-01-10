GOROOT := /usr/local/go
GOPATH := $(shell pwd)
GOBIN  := $(GOPATH)/bin
PATH   := $(GOROOT)/bin:$(PATH)
DEPS   := github.com/gin-gonic/gin github.com/tarm/serial github.com/mattn/go-isatty

all: server

deps: $(DEPS)
	GOPATH=$(GOPATH) go get -u $^

server: main.go modem.go config.go
    # always format code
		GOPATH=$(GOPATH) go fmt $^
    # binary
		GOPATH=$(GOPATH) go build -o $@ -v $^
		touch $@

windows: main.go modem.go config.go
    # always format code
		GOPATH=$(GOPATH) go fmt $^
    # binary
		GOOS=windows GOARCH=amd64 GOPATH=$(GOPATH) go build -o server.exe -v $^
		touch server.exe

.PHONY: $(DEPS) clean

clean:
	rm -f server server.exe
