# Disable built-in rules
MAKEFLAGS += --no-builtin-rules
.SUFFIXES:

include tools/Makefile/lib.mk

vendors := vendor
# Tracking .git/HEAD allows to recompile things
global-deps := $(MAKEFILE_LIST) .git/HEAD
clean      := .cache
distclean  := $(vendors)
docker/dev/image := sqreen/go-agent-dev
docker/dev/container := go-agent-dev
docker/dev/container/options := -e GO111MODULE=on -e GOCACHE=$$PWD/.cache -e GOPATH=$$PWD/.cache/go
docker/dev/container/dockerfile := tools/docker/dev/Dockerfile
go/target := $(shell go env GOOS)_$(shell go env GOARCH)
agent/library/static := pkg/$(go/target)/sqreen/agent.a
protobufs := $(patsubst %.proto,%.pb.go,$(shell find agent -name '*.proto'))
ginkgo/flags := -r --randomizeAllSpecs --randomizeSuites --progress -p
protoc/flags := -I. -Ivendor --gogo_out=google/protobuf/any.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types:.

define dockerize =
if $(lib/docker/is_in_container); then $(lib/argv/1); else docker exec -i $(docker/dev/container) bash -c "$(lib/argv/1)"; fi
endef

##
# Helper variables to make easier reading the rules requiring docker images or
# containers.
#
dev-container := .cache/docker/dev/run
dev-image := .cache/docker/dev/build
needs-dev-container := $(if $(shell $(lib/docker/is_in_container) && echo y),,$(dev-container))
needs-dev-image := $(if $(shell $(lib/docker/is_in_container) && echo y),,$(dev-image))
needs-protobufs := $(protobufs)
needs-vendors := .cache/go/vendor

#-----------------------------------------------------------------------------
# General
#------------------------------------------------------------------------------

.PHONY: help
help:
	@echo Targets: $(help)

.PHONY: all
all: $(agent/library/static)
help += all

.PHONY: clean
clean:
	rm -rf $(clean)
	docker rm -f $(docker/dev/container) || true
	docker rmi -f $(docker/dev/image) || true
help += clean

.PHONY: distclean
distclean: clean
	rm -rf $(distclean)
help += distclean

#-----------------------------------------------------------------------------
# Library
#------------------------------------------------------------------------------

$(agent/library/static): $(needs-dev-container) $(needs-protobufs) $(needs-vendors)
	$(call dockerize, go install -v agent)

#-----------------------------------------------------------------------------
# Tests
#------------------------------------------------------------------------------

.PHONY: test
test: $(needs-dev-container) $(needs-vendors) $(needs-protobufs)
	$(call dockerize, ginkgo $(ginkgo/flags) ./agent)

.PHONY: test-coverage
test-coverage: $(needs-dev-container) $(needs-vendors) $(needs-protobufs)
	$(call dockerize, ginkgo $(ginkgo/flags) -cover -coverprofile=coverage.txt ./agent)

.PHONY: test-race
test-race: $(needs-dev-container) $(needs-vendors) $(needs-protobufs)
	$(call dockerize, ginkgo $(ginkgo/flags) -race ./agent)

#-----------------------------------------------------------------------------
# Vendor directory
#-----------------------------------------------------------------------------

$(needs-vendors): go.mod $(global-deps) $(needs-dev-container)
	$(call dockerize, go mod vendor)
	mkdir -p $(@D) && touch $@

.PHONY: vendor
vendor: $(needs-vendors)

.PHONY: .revendor
.revendor:
	$(call dockerize, go mod vendor)

#-----------------------------------------------------------------------------
# Protocol buffers
#-----------------------------------------------------------------------------

%.pb.go: %.proto $(needs-dev-container) $(needs-vendors)
	$(call dockerize, protoc $(protoc/flags) $<)
	make .revendor

#-----------------------------------------------------------------------------
# Dockerized dev environment
#------------------------------------------------------------------------------

.PHONY: shell
shell: $(needs-dev-container)
	docker exec -it $(docker/dev/container) bash
help += shell

$(dev-image): $(docker/dev/container/dockerfile) $(global-deps)
	docker build -t $(docker/dev/image) --build-arg uid=$(shell id -u) -f $< .
	mkdir -p $(@D) && touch $@

$(dev-container): $(needs-dev-image)
	$(call lib/docker/is_container_running, $(docker/dev/container)) && docker rm -f $(docker/dev/container) || true
	docker run $(docker/dev/container/options) -ditv $$PWD:$$PWD -w $$PWD $(docker/dev/container/options) --name $(docker/dev/container) $(docker/dev/image)
	mkdir -p $(@D) && touch $@
