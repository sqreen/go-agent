#
# Build a Docker image able to run the Makefile.
# This is a different approach than using the Dockerfile to build a project.
# We do not want Dockerfiles to substitute Makefiles. It also has the advantage
# to make possible compiling "natively" without docker.
#

FROM debian:sid

# set noninteractive front-end during the build
ARG DEBIAN_FRONTEND=noninteractive

# lang settings
RUN apt-get -q update \
    && apt-get -q install -y --no-install-recommends locales && locale-gen C.UTF-8
ENV LANG C.UTF-8
ENV LC_ALL C.UTF-8

RUN apt-get -q update && \
    apt-get -q install -y --no-install-recommends \
    ca-certificates \
    libc-dev \
    git \
    make \
    gdbserver \
    golang-1.11 \
    gcc \
    libc-dev \
    sudo \
    curl \
    unzip

ENV PATH /usr/lib/go-1.11/bin:$PATH

# Protocol Buffers Compiler
ARG protocversion=3.6.1
RUN v=$protocversion && m=$(uname -m) && curl -sSL https://github.com/protocolbuffers/protobuf/releases/download/v$v/protoc-$v-linux-$m.zip > /tmp/protoc-$v-linux-$m.zip \
    && unzip /tmp/protoc-$v-linux-$m.zip -d /opt/protoc-$v \
    && chmod -R a+rx /opt/protoc-$v
ENV PATH /opt/protoc-$protocversion/bin:$PATH

# Additional go tools
RUN GOBIN=/usr/local/bin go get -u -v github.com/gogo/protobuf/protoc-gen-gogo \
                                      github.com/onsi/ginkgo/ginkgo

# Create a non-root user with the same uid as on the host to allow proper file
# permissions created from the container in volumes, and to Since it is not root, allow
# calling sudo without password when required.
ARG uid=10000
RUN useradd -M --uid $uid --user-group devuser \
    && echo 'devuser ALL=(ALL) NOPASSWD:ALL' >> /etc/sudoers.d/devuser \
    && echo 'Defaults exempt_group+=user' >> /etc/sudoers.d/devuser \
    && chmod a=r,o= /etc/sudoers.d/devuser
USER devuser

ENTRYPOINT bash