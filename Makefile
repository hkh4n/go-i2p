RELEASE_VERSION=0.0.1
RELEASE_DESCRIPTION=`cat PASTA.md`
REPO := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

ifdef GOROOT
	GO = $(GOROOT)/bin/go
endif

GO ?= $(shell which go)

ifeq ($(GOOS),windows)
	EXE := $(REPO)/go-i2p.exe
else
	EXE := $(REPO)/go-i2p
endif

build: clean $(EXE)

$(EXE):
	$(GO) build -v -o $(EXE)

test: fmt
	$(GO) test -vv -failfast ./lib/common/...

clean:
	$(GO) clean -v

fmt:
	find . -name '*.go' -exec gofmt -w -s {} \;

info:
	echo "GOROOT: ${GOROOT}"
	echo "GO: ${GO}"
	echo "REPO: ${REPO}"

release:
	github-release release -u go-i2p -repo go-i2p -name "${RELEASE_VERSION}" -d "${RELEASE_DESCRIPTION}" -p