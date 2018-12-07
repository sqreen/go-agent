include sdk/Makefile/lib.mk

# Tracking .git/HEAD allows to recompile things
globaldeps := $(MAKEFILE_LIST) .git/HEAD
clean      := .cache
distclean  :=
docker/sdk/image := sqreen/go-agent-sdk
docker/sdk/container := go-agent-sdk
docker/sdk/container/options := -e GOPATH=$$PWD -e GOCACHE=$$PWD/.cache
docker/sdk/container/dockerfile := sdk/docker/dev/Dockerfile
go/target := $(shell go env GOOS)_$(shell go env GOARCH)
agent/library/static := pkg/$(go/target)/sqreen/agent.a

define dockerize =
$(lib/docker/is_in_container) && $(lib/argv/1) || docker exec -i $(docker/sdk/container) $(lib/argv/1)
endef

##
# Helper variables to make easier reading the rules requiring docker images or
# containers.
#
sdk-container := .cache/docker/dev/run
sdk-image := .cache/docker/dev/build
needs-sdk-container := $(if $(shell $(lib/docker/is_in_container) && echo y),,$(sdk-container))
needs-sdk-image := $(if $(shell $(lib/docker/is_in_container) && echo y),,$(sdk-image))

.PHONY: help
help:
	@echo Targets: $(help)

.PHONY: all
all: $(agent/library/static)
help += all

$(agent/library/static): $(needs-sdk-container)
	$(call dockerize, go install -v sqreen/agent)

.PHONY: clean
clean:
	rm -rf .cache
	docker rm -f $(docker/sdk/container) || true
	docker rmi -f $(docker/sdk/image) || true
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
	$(call lib/docker/is_container_running, $(docker/sdk/container)) && docker rm -f $(docker/sdk/container) || true
	docker run $(docker/sdk/container/options) -ditv $$PWD:$$PWD -w $$PWD $(docker/sdk/container/options) --name $(docker/sdk/container) $(docker/sdk/image)
	mkdir -p $(@D) && touch $@
