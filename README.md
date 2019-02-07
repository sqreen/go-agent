![Sqreen](https://s3-eu-west-1.amazonaws.com/sqreen-assets/npm/20171113/sqreen_horizontal_250.png)

# [Sqreen](https://www.sqreen.com/)'s Security Agent for Go

Sqreen monitors your application security and helps you easily protect it from
common vulnerabilities or advanced attacks.

- Gain visibility into your application security.
- One-click protection from common vulnerabilities.
- Easily enforce custom protection rules into your app.
- Identify malicious users before they cause harm.
- Integrate with your workflow.

![Dashboard](https://d33wubrfki0l68.cloudfront.net/0fe441513f505601d03b25249deddd8fd1eb2a49/e2da6/img/new/illustrations/dashboard-mockup.png)

Sqreen also protects applications against common security threats such as
database injections, cross-site scripting attacks, scans, or authentication
activity inside the application to detect and block account takeover attacks. It
monitors functions in the application (I/O, authentication, network, command
execution, etc.) and provides dedicated security logic at run-time.

For more details, visit [sqreen.com](https://www.sqreen.com/)

# Installation

1. Download the Go agent and the SDK using `go get`:

    ```sh
    $ go get github.com/sqreen/go-agent/...
    ```

1. Import the package `agent` in your `main` package of your app:

    ```go
    import _ "github.com/sqreen/go-agent/agent"
    ```

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

1. Set up Sqreen's middleware functions according to the web framework you use:
   - [sqhttp](https://godoc.org/github.com/sqreen/go-agent/sdk/middleware/sqhttp) for the standard net/http package.
   - [Gin](https://godoc.org/github.com/sqreen/go-agent/sdk/middleware/sqgin)
   - [Echo](https://godoc.org/github.com/sqreen/go-agent/sdk/middleware/sqecho)
   - Coming soon: [Iris](https://github.com/sqreen/go-agent/pull/22),
     [gRPC](https://github.com/sqreen/go-agent/pull/23) (please upvote if
     interested ;)).
   
   If your framework is not in the list, it is usually possible to use the
   standard `net/http` middleware. If not, please open an issue in this
   repository to start a discussion about it.

1. Optionally, use the [SDK](https://godoc.org/github.com/sqreen/go-agent/sdk)
   to send security [events related to
   users](https://godoc.org/github.com/sqreen/go-agent/sdk#HTTPRequestRecord.ForUser)
   (eg. signing-in) or completely [custom security-related
   events](https://godoc.org/github.com/sqreen/go-agent/sdk#HTTPRequestRecord.TrackEvent)
   you would like to track (eg. password changes).

# Licensing

Sqreen for Go is free-to-use, proprietary software.

# Terms

Copyright (c) 2019 Sqreen. All Rights Reserved. Please refer to our terms for
more information: https://www.sqreen.com/terms.html
