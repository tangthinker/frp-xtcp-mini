export PATH := $(PATH):`go env GOPATH`/bin
export GO111MODULE=on
LDFLAGS := -s -w

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
LIBFRPC_EXT := dylib
else ifeq ($(UNAME_S),Linux)
LIBFRPC_EXT := so
else
LIBFRPC_EXT := dll
endif

.PHONY: frps frpc libfrpc

all: env fmt build

build: frps frpc

env:
	@go version

fmt:
	go fmt ./...

fmt-more:
	gofumpt -l -w .

gci:
	gci write -s standard -s default -s "prefix(github.com/fatedier/frp/)" ./

vet:
	go vet ./...

frps:
	env CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -tags "frps" -o bin/frps ./cmd/frps

frpc:
	env CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -tags "frpc" -o bin/frpc ./cmd/frpc

libfrpc:
	env CGO_ENABLED=1 go build -trimpath -ldflags "$(LDFLAGS)" -buildmode=c-shared -o bin/libfrpc.$(LIBFRPC_EXT) ./cmd/libfrpc
	cp libfrpc/frpc.h bin/frpc.h

test: gotest

gotest:
	go test -v --cover ./cmd/frpc/... ./cmd/frps/...
	go test -v --cover ./client/...
	go test -v --cover ./server/...
	go test -v --cover ./pkg/...
	go test -v --cover ./libfrpc/...

e2e:
	./hack/run-e2e.sh

e2e-trace:
	DEBUG=true LOG_LEVEL=trace ./hack/run-e2e.sh

alltest: vet gotest e2e

clean:
	rm -f ./bin/frpc
	rm -f ./bin/frps
	rm -f ./bin/libfrpc.dylib ./bin/libfrpc.so ./bin/libfrpc.dll ./bin/libfrpc.h ./bin/frpc.h
	rm -rf ./lastversion
	rm -rf ./.cache
	rm -rf ./.compat
