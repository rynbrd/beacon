name=beacon
version=$(shell git describe --tags --dirty)

gopkgs=./cmd/beacon ./beacon ./container ./docker ./etcd

export GOBIN=$(shell pwd)/bin
export GOPATH=$(shell pwd)/.go
org_url=github.com/BlueDragonX
org_path=$(GOPATH)/src/$(org_url)
project_url=$(org_url)/$(name)
project_path=$(org_path)/$(name)

.PHONY: clean test static
all: build
build: $(GOBIN)/beacon
static: $(OGBIN)/beacon.static

clean:
	rm -rf $(GOBIN) $(GOPATH)

$(project_path):
	mkdir -p $(org_path)
	ln -sf $(shell pwd) $(project_path)

$(GOBIN)/beacon: $(project_path)
	go get -d $(project_url)/cmd/beacon
	go install $(project_url)/cmd/beacon

$(GOBIN)/beacon.static: $(project_path)
	go get -d $(project_url)/cmd/beacon
	GOBIN=$(GOPATH)/bin go install -ldflags "-linkmode external -extldflags -static" $(project_url)/cmd/beacon
	mkdir -p $(GOBIN)
	mv $(GOPATH)/bin/beacon $(GOBIN)/beacon.static

test: $(project_path)
	test -z "$(shell gofmt -s -l $(gopkgs))"
	go vet $(gopkgs)
	go get -d -t $(gopkgs)
	go test -v -race $(gopkgs)
