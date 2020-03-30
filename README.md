<p align="center">
<img width="40%" src="doc/images/sqreen-gopher.png" alt="Sqreen for Go" title="Sqreen for Go" />
</p>

# [Sqreen](https://www.sqreen.com/)'s Application Security Management for Go

[![Release](https://img.shields.io/github/release/sqreen/go-agent.svg)](https://github.com/sqreen/go-agent/releases)
[![GoDoc](https://godoc.org/github.com/sqreen/go-agent?status.svg)](https://godoc.org/github.com/sqreen/go-agent)
[![Go Report Card](https://goreportcard.com/badge/github.com/sqreen/go-agent)](https://goreportcard.com/report/github.com/sqreen/go-agent)
[![Build Status](https://dev.azure.com/sqreenci/Go%20Agent/_apis/build/status/sqreen.go-agent?branchName=master)](https://dev.azure.com/sqreenci/Go%20Agent/_build/latest?definitionId=8&branchName=master)
[![Sourcegraph](https://sourcegraph.com/github.com/sqreen/go-agent/-/badge.svg)](https://sourcegraph.com/github.com/sqreen/go-agent?badge)

After performance monitoring (APM), error and log monitoring it’s time to add a
security component into your app. Sqreen’s microagent automatically monitors
sensitive app’s routines, blocks attacks and reports actionable infos to your
dashboard.

![Dashboard](https://sqreen-assets.s3-eu-west-1.amazonaws.com/miscellaneous/dashboard.gif)

Sqreen provides automatic defense against attacks:

- Protect with security modules: RASP (Runtime Application Self-Protection),
  in-app WAF (Web Application Firewall), Account takeovers and more.

- Sqreen’s modules adapt to your application stack with no need of configuration.

- Prevent attacks from the OWASP Top 10 (Injections, XSS and more), 0-days,
  data Leaks, and more.
  
- Create security automation playbooks that automatically react against
  your advanced business-logic threats.

For more details, visit [sqreen.com](https://www.sqreen.com/)

# Quick start

1. Use the middleware function for the Go web framework you use:
   - [sqhttp](https://godoc.org/github.com/sqreen/go-agent/sdk/middleware/sqhttp) for the standard `net/http` package.
   - [Gin](https://godoc.org/github.com/sqreen/go-agent/sdk/middleware/sqgin) for `github.com/gin-gonic/gin`.
   - [Echo](https://godoc.org/github.com/sqreen/go-agent/sdk/middleware/sqecho) for `github.com/labstack/echo`.

   If your framework is not listed, it is usually possible to use instead the
   standard `net/http` middleware. If not, please, let us know by [creating an
   issue](http://github.com/sqreen/go-agent/issues/new).

1. Without Go modules: Download the new dependencies

   `go get` will automatically download the new dependencies of the SDK, including
   Sqreen's agent for Go:

   ```consol
   $ go get -d -v ./...
   ```

1. Compile your program with Sqreen

   Sqreen's dynamic configuration of your protection is made possible thanks to
   Go instrumentation. It is safely performed at compilation time by the following
   instrumentation tool.

   Install the following instrumentation tool and compile your program using it in
   order to enable Sqreen.

   1. Use `go install` to compile the instrumentation tool:

      ```console
      $ go install github.com/sqreen/go-agent/sdk/sqreen-instrumentation
      ```

      By default, the resulting `sqreen-instrumentation` tool is installed in the
      `bin` directory of the `GOPATH`. You can find it using `go env GOPATH`.

   1. Configure the Go toolchain to use it:

      Use the instrumentation tool using the go options
      `-a -toolexec /path/to/sqreen-instrumentation`.

      It can be done either in your Go compilation command lines or by setting the
      `GOFLAGS` environment variable.
      
      For example, the following two commands are equivalent:
      ```console
      $ go build -a -toolexec $(go env GOPATH)/bin/sqreen-instrumentation my-project
      $ env GOFLAGS="-a -toolexec $(go env GOPATH)/bin/sqreen-instrumentation" go build my-project
      ```
    
1. [Signup to Sqreen](https://my.sqreen.io/signup) to get a token for your app,
   and store it in the agent's configuration file `sqreen.yaml`:
   
    ```sh
    app_name: Your Go service name
    token: your token
    ```
   
   This file can be stored in your current working directory when starting the
   executable, the same directory as your app's executable file, or in any other
   path by defining the configuration file location into the environment
   variable `SQREEN_CONFIG_FILE`.

1. You are done!  
   Just recompile your Go program and the go toolchain will download the latests
   agent version.

1. Optionally, use the [SDK](https://godoc.org/github.com/sqreen/go-agent/sdk)
   to perform [user monitoring](https://godoc.org/github.com/sqreen/go-agent/sdk#HTTPRequestRecord.ForUser)
   (eg. signing-in) or [custom security events](https://godoc.org/github.com/sqreen/go-agent/sdk#HTTPRequestRecord.TrackEvent)
   you would like to track (eg. password changes).

Find out more about the agent setup at https://docs.sqreen.com/go/installation/

# Licensing

Sqreen for Go is free-to-use, proprietary software.

# Terms

Copyright (c) 2019 Sqreen. All Rights Reserved. Please refer to our terms for
more information: https://www.sqreen.com/terms.html
