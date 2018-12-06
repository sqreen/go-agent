include sdk/Makefile/lib.mk

# Tracking .git/HEAD allows to recompile things
globaldeps := $(MAKEFILE_LIST) .git/HEAD
clean      := .cache
distclean  :=
docker/sdk/image := sqreen/go-agent-sdk
docker/sdk/container := go-agent-sdk
docker/sdk/container/options := -e GOCACHE=$$PWD/.cache
docker/sdk/container/dockerfile := sdk/docker/dev/Dockerfile
go/target := $(shell go env GOOS)_$(shell go env GOARCH)

agent/library/static := pkg/$(go/target)/sqreen/protobuf/api.a

##
# Helper variables to make easier reading the rules requiring docker images or
# containers.
#
sdk-container := .cache/docker/dev/run
sdk-image := .cache/docker/dev/build
needs-sdk-container := $(sdk-container)
needs-sdk-image := $(sdk-image)

.PHONY: help
help:
	@echo Targets: $(help)

.PHONY: all
all: 

$(agent/library/static):
	go install -v sqreen/agent

.PHONY: clean
clean:
	rm -rf .cache
	docker rm -f $(docker/sdk/container)
	docker rmi -f $(docker/sdk/image)
help += clean

#------------------------------------------------------------------------------
# Dockerized dev environment
#------------------------------------------------------------------------------

.PHONY: shell
shell: $(needs-sdk-container)
	docker exec -it $(docker/sdk/container) bash
help += shell

$(sdk-image): $(docker/sdk/container/dockerfile) $(globaldeps)
	docker build -t $(docker/sdk/image) --build-arg uid=$(shell id -u) -f $< .
	mkdir -p $(@D) && touch $@

$(sdk-container): $(needs-sdk-image)
	$(call lib/docker/is_container_running, $(docker/sdk/container)) && docker rm -f $(docker/sdk/container)
	docker run $(docker/sdk/container/options) -ditv $$PWD:$$PWD -w $$PWD $(docker/sdk/container/options) --name $(docker/sdk/container) $(docker/sdk/image)
	mkdir -p $(@D) && touch $@
