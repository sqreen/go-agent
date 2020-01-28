FROM golang:1 AS dev
WORKDIR /app
# Install binutils for ldd
RUN apt update && apt install binutils
COPY . .
RUN go install -v github.com/sqreen/go-agent/sdk/sqreen-instrumentation-tool
RUN go build -v -ldflags="-s -w" -a -toolexec /go/bin/sqreen-instrumentation-tool -o bin/api api/main.go
# Use ldd to list the shared libraries and copy them into deps/
RUN ldd bin/api | tr -s '[:blank:]' '\n' | grep '^/' | \
    xargs -I % sh -c 'mkdir -p $(dirname deps%); cp % deps%;'

# Example of a FROM scratch target
FROM scratch AS prod-scratch
# Empty image so we need to copy the binary and its shared libraries.
COPY --from=dev /app/deps /app/bin/api /
ENTRYPOINT [ "/api" ]

# Example of a FROM alpine target
FROM alpine AS prod-alpine
# Every expected shared library is present, but the C library doesn't have the
# expected name. The libc6-compat package adds symlinks with the expected
# names.
RUN apk update && apk add libc6-compat
# Now only copy the binary file.
COPY --from=dev /app/bin/api /
ENTRYPOINT [ "/api" ]

# Example of a FROM debian target
FROM debian AS prod-debian
# Only copy the binary file since the required shared libaries are present
# by default.
COPY --from=dev /app/bin/api /
ENTRYPOINT [ "/api" ]