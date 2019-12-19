![Sqreen](https://sqreen-assets.s3-eu-west-1.amazonaws.com/logos/sqreen-logo-264-1.svg)

# [Sqreen](https://www.sqreen.com/)'s Application Security Management for Go

[![Release](https://img.shields.io/github/release/sqreen/go-agent.svg)](https://github.com/sqreen/go-agent/releases)
[![GoDoc](https://godoc.org/github.com/sqreen/go-agent?status.svg)](https://godoc.org/github.com/sqreen/go-agent)
[![Go Report Card](https://goreportcard.com/badge/github.com/sqreen/go-agent)](https://goreportcard.com/report/github.com/sqreen/go-agent)
[![codecov](https://codecov.io/gh/sqreen/go-agent/branch/master/graph/badge.svg)](https://codecov.io/gh/sqreen/go-agent)
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

# Installation

Installing the agent is just installing a regular Go package. Simply import it
and let the toolchain do the job.


1. Import the agent package into the `main` package of your app:

    ```go
    import _ "github.com/sqreen/go-agent/agent"
    ```

1. Use the middleware function for the Go web framework you use:
   - [sqhttp](https://godoc.org/github.com/sqreen/go-agent/sdk/middleware/sqhttp) for the standard `net/http` package.
   - [Gin](https://godoc.org/github.com/sqreen/go-agent/sdk/middleware/sqgin) for `github.com/gin-gonic/gin`.
   - [Echo](https://godoc.org/github.com/sqreen/go-agent/sdk/middleware/sqecho) for `github.com/labstack/echo`.
   
   If your framework is not in the list, it is usually possible to use the
   standard `net/http` middleware. If not, please, [create an issue](http://github.com/sqreen/go-agent/issues/new)
   to let us know.

1. [Signup to Sqreen](https://my.sqreen.io/signup) to get a token for your app,
   and store it in the agent's configuration file `sqreen.yaml`:
   
    ```sh
    token: your token
    app_name: Your App Name
    ```
   
   This file can be stored in your current working directory when starting the
   executable, the same directory as your app's executable file, or in any other
   path by defining the configuration file location into the environment
   variable `SQREEN_CONFIG_FILE`.

1. You are done!  
   Just recompile your Go program and the go toolchain will download the latests
   agent version.

1. Optionally, use the [SDK](https://godoc.org/github.com/sqreen/go-agent/sdk)
   to send security [events related to
   users](https://godoc.org/github.com/sqreen/go-agent/sdk#HTTPRequestRecord.ForUser)
   (eg. signing-in) or completely [custom security-related
   events](https://godoc.org/github.com/sqreen/go-agent/sdk#HTTPRequestRecord.TrackEvent)
   you would like to track (eg. password changes).

Find out more about the agent installation at https://docs.sqreen.com/go/installation/

# Licensing

Sqreen for Go is free-to-use, proprietary software.

# Terms

Copyright (c) 2019 Sqreen. All Rights Reserved. Please refer to our terms for
more information: https://www.sqreen.com/terms.html
