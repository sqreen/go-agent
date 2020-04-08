# Build image
FROM golang:1.13 AS build

# Workdir out of the GOPATH to enable the Go modules mode.
WORKDIR /app
COPY . .

# Installing Sqreen's instrumentation tool.
# Go modules make it easier by correctly choosing the version of your go.mod.
RUN go build -v github.com/sqreen/go-agent/sdk/sqreen-instrumentation-tool

# Compile the app with the previously built tool.
RUN go build -v -a -toolexec $PWD/sqreen-instrumentation-tool -o app .

# Application image with Sqreen
FROM gcr.io/distroless/base:latest
COPY --from=build /app/app /usr/local/bin/app
EXPOSE 8080
CMD ["/usr/local/bin/app"]