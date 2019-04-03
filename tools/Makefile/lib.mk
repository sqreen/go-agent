##
# lib/docker/is_container_running(container, host?)
# Exit status is 0 (true) when given `container` state is running.
#
define lib/docker/is_container_running =
[ "$$(docker $(strip $2) inspect -f '{{.State.Running}}' $(lib/argv/1) 2>/dev/null)" = true ]
endef

##
# lib/docker/is_make_in_docker()
# Return True (non-empty string) when make is inside a container.
#
define lib/docker/is_in_container =
test -f /.dockerinit -o -f /.dockerenv
endef

##
# Mandatory argument getters.
#
lib/argv/1  = $(call lib/argv, $1)
lib/argv/2  = $(call lib/argv, $2)
lib/argv/3  = $(call lib/argv, $3)
lib/argv/4  = $(call lib/argv, $4)
lib/argv/5  = $(call lib/argv, $5)
lib/argv/6  = $(call lib/argv, $6)
lib/argv/7  = $(call lib/argv, $7)
lib/argv/8  = $(call lib/argv, $8)
lib/argv/9  = $(call lib/argv, $9)
lib/argv/10 = $(call lib/argv, $(10))
define lib/argv =
$(or $(strip $1), $(error missing mandatory argument))
endef

##
# git/ref/head()
# Return the short commit reference to HEAD.
#
git/ref/head = $(shell git rev-parse --short HEAD)
