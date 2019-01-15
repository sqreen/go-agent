![Sqreen](https://s3-eu-west-1.amazonaws.com/sqreen-assets/npm/20171113/sqreen_horizontal_250.png)

# [Sqreen](https://www.sqreen.com/)'s Security Agent for Go

Sqreen agent monitors functions in the application (I/O, authentication,
network, command execution, etc.) and provides dedicated security logic at
run-time.

Sqreen protects applications against common security threats.

Here are some security events which can be blocked and reported:
* Database injection (SQL/NoSQL).
* Cross-site scripting attack.
* Significant bad bot / scan activity against the application (scans which
  require attention).
* Peak of HTTP errors (40x, 50x) related to security activity against the
  application.
* Targeted (human) investigation led against your application.
* New vulnerabilities detected in a third-party modules used by the application.
* Authentication activity inside the application to detect and block account
  takeover attacks.

For more details, visit [sqreen.com](https://www.sqreen.com/)

# Installation

1. Download the Go agent and the SDK using `go get`:

    ```sh
    $ go get github.com/sqreen/go-agent/...
    ```

1. Import package `agent` in your `main` package of your app:

    ```go
    import _ "github.com/sqreen/go-agent/agent"
    ```
1. [Signup to Sqreen](https://my.sqreen.io/signup) to get a token for your app,
   and store it in the agent's configuration file:

    ```sh
    $ cat sqreen.yaml
    token: "your token"
    ```
    
   This file needs to be stored in the same directory as your app's executable
   file, as the agent will look for it in the current working directory.

# Licensing

Sqreen for Go is free-to-use, proprietary software.

# Terms

Copyright (c) 2019 Sqreen. All Rights Reserved. Please refer to our terms for
more information: https://www.sqreen.com/terms.html
