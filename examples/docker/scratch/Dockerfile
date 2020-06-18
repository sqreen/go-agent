# Example of a multi-stage dockerfile building the example with Sqreen, and
# creating a final scratch docker image. This example is special as the scratch
# image is absolutely empty and more files need to be copied in that case.

# Build docker image
ARG GO_VERSION=1
FROM golang:$GO_VERSION AS build
# Workdir out of the GOPATH to enable the Go modules mode.
WORKDIR /app
COPY . .

# Install Sqreen's instrumentation tool.
# Go modules make it easier by correctly choosing the version of your go.mod.
RUN go build -v github.com/sqreen/go-agent/sdk/sqreen-instrumentation-tool

# Compile the app with the previously built tool.
RUN go build -v -a -toolexec $PWD/sqreen-instrumentation-tool -o hello-sqreen .

# Now prepare a directory with the shared libraries the compiled program file
# requires by using ldd:
# 1. Install binutils for ldd
RUN apt update && apt install binutils ca-certificates
# 2. Use ldd to list the shared libraries and copy them into deps/
RUN ldd hello-sqreen | tr -s '[:blank:]' '\n' | grep '^/' | \
    xargs -I % sh -c 'mkdir -p $(dirname deps%); cp % deps%;'

# Final application docker image
FROM scratch
# The scratch image is empty so we need to copy the binary and its shared libraries.
COPY --from=build /app/deps /app/hello-sqreen /
# Also copy the CA certificates required by the HTTPS connection to Sqreen.
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
EXPOSE 8080
ENTRYPOINT [ "/hello-sqreen" ]