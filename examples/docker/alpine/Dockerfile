# Example of a multi-stage dockerfile building the example with Sqreen, and
# creating a final alpine docker image.

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

# Final application docker image
FROM alpine
# Copy the app program file
COPY --from=build /app/hello-sqreen /usr/local/bin
# Every required shared library is present, but the C library doesn't have the
# correct name on alpine. The libc6-compat package adds a symlinks with the
# expected name.
# Also add the CA certificates required by the HTTPS connection to Sqreen.
RUN apk update && apk add libc6-compat ca-certificates
EXPOSE 8080
ENTRYPOINT [ "/usr/local/bin/hello-sqreen" ]
