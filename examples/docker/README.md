# Docker image examples

This example allows to create a docker image of Go web server with Sqreen.

## Examples

The following Dockerfile examples show-case multi-stage docker builds in order
to separate the build environment and the tools it needs from the end
application image.

They all use the official [golang docker image](https://hub.docker.com/_/golang)
which contains by default everything required to compile a Go program with
Sqreen.

### Building the image examples

#### Debian docker image example

The Dockerfile of this example can be found in the `debian/` subdirectory.
It shows how to produce a debian docker image with the Go web server
protected by Sqreen.

Build the docker image and tag it with the image name `hello-sqreen` by doing:

```console
examples/docker $ docker build -t hello-sqreen:debian -f debian/Dockerfile .
```

#### Alpine image

The Dockerfile of this example can be found in the `alpine/` subdirectory.
It shows how to produce an alpine docker image with the Go web server
protected by Sqreen.

Build the docker image and tag it with the image name `hello-sqreen` by doing:

```console
examples/docker $ docker build -t hello-sqreen -f alpine/Dockerfile .
```

#### Scratch image

The Dockerfile of this example can be found in the `scratch/` subdirectory.
It shows how to produce a docker from scratch with the Go web server protected
by Sqreen.

Build the docker image and tag it with the image name `hello-sqreen` by doing:

```console
examples/docker $ docker build -t hello-sqreen -f alpine/Dockerfile .
```

### Running the build docker image with Sqreen

Once you have built your `hello-sqreen` docker image by following one of the
previous docker build examples, you can then run it.

1. Get your Sqreen credentials from our dashboard at https://my.sqreen.com/new-application#golang-agent

1. Run the docker image with Sqreen by at least passing the Sqreen application
   token:
   ```console
   examples/docker $ docker run -t -p 8080:8080 -e SQREEN_TOKEN="oh my token" --rm hello-sqreen
   ```
   See the [configuration](https://docs.sqreen.com/go/configuration/) for the
   full list of configuration options.

Congratulations, your are running a Go web server now protected by Sqreen!

<p align="center">
<img width="60%" src="../../doc/images/blocking-page-with-gopher.png" alt="Sqreen for Go" title="Sqreen for Go" />
</p>
