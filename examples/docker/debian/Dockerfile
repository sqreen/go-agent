# Example of a multi-stage dockerfile building the example with Sqreen, and
# creating a final debian docker image.

# Build docker image
ARG GO_VERSION=1
FROM golang:$GO_VERSION AS build
# Workdir out of the GOPATH to enable the Go modules mode.
WORKDIR /app
COPY . .

# Update the go.mod and go.sum dependencies
RUN go get -d github.com/sqreen/go-agent/sdk/sqreen-instrumentation-tool
RUN go get -d ./...

# Install Sqreen's instrumentation tool.
RUN go build -v github.com/sqreen/go-agent/sdk/sqreen-instrumentation-tool

# Compile the app with the previously built tool.
RUN go build -v -a -toolexec $PWD/sqreen-instrumentation-tool -o hello-sqreen .

# Final application docker image
FROM debian:stable-slim
# Copy the app program file
COPY --from=build /app/hello-sqreen /usr/local/bin
# Add the CA certificates required by the HTTPS connection to Sqreen.
RUN apt update && apt install -y ca-certificates
EXPOSE 8080
ENTRYPOINT [ "/usr/local/bin/hello-sqreen" ]
