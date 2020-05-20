# Example of a multi-stage dockerfile building the example with Sqreen, and
# creating a final debian docker image.

# Build docker image
FROM golang:1 AS build
# Workdir out of the GOPATH to enable the Go modules mode.
WORKDIR /app
COPY . .

# Install Sqreen's instrumentation tool.
# Go modules make it easier by correctly choosing the version of your go.mod.
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
