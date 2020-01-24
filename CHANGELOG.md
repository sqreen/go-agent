# v0.1.0-beta.10

## Breaking Change

- (#89) Go instrumentation: Sqreen's dynamic configuration of the protection
  your Go programs is made possible at run time thanks to Go instrumentation.
  It is a building block of the upcoming run time self-protection (aka RASP) and
  it is safely performed at compilation time by an instrumentation tool that
  seamlessly integrates with the Go toolchain. To begin with, only a specific
  set of Go packages are instrumented: the agent and `database/sql` (to prepare
  the upcoming SQL injection protection).

  Please, find out how to install and use the tool on the new agent installation
  documentation available at https://docs.sqreen.com/go/installation/.

## New Features

- (#90) The SDK now imports the agent package to no longer have to import it in the
  `main` package. The SDK is indeed mandatory when setting up Sqreen for Go,
  making it the best place to import the agent.

- (#91) The program dependencies are now sent to Sqreen to perform dependency
  analysis (outdated, vulnerable, etc.). They are only available when the Go
  program you compile is a Go module. Sqreen's dashboard Dependency page will
  made available be soon.

## Fix

- (#92) Vendoring using `go mod vendor` could lead to compilation errors due to
  missing files.

# v0.1.0-beta.9

## New Features

- Request parameters such as query or post parameters are now added in the attack
  events and shown in the attack logs and in the event explorer pages of our dashboard. (#84)

- PII scrubbing is now performed on every event sent to Sqreen, as documented
  on https://docs.sqreen.com/guides/how-sqreen-works/#pii-scrubbing. (#86)

## Fixes

- Add PII scrubbing to the WAF logs that may include data from the request. (#87)

## Internal Changes

- The In-App WAF has been intensively optimized so that large requests can no longer impact
  its execution time. (#83)

# v0.1.0-beta.8

## Internal Changes

- In-App WAF:
  - Dynamically set the WAF timeout (#79).
  - Ignore WAF timeout errors and add more context when reporting an error (#80).
  - Update the libsqreen to v0.4.0 to add support for the `@pm` operator.

# v0.1.0-beta.7

## Breaking Changes

- CGO bindings are now involved in the compilation of the agent and will require
  the `gcc` compiler and the C library headers. Note that compiling the agent
  without CGO (`CGO_ENABLED=0`) is still possible but will disable some agent
  features; in this case the new WAF feature described below.

## New Feature

- Add support for the In-App WAF: an out-of-the-box Web-Application Firewall
  leveraging the full application context, that is fail-safe, has limited false
  positives and won’t require heavy fine-tuning. Only darwin/amd64 and
  linux/amd64 targets are supported so far. Any other target will get this
  feature disabled. More targets will be added in future versions. (#77)

## Minor Change

- Increase the internal timeout value of the HTTP client to Sqreen's backend in
  order to be more resilient to normal networking delays.

## Fix

- Fix a compilation error on 32-bit target architectures.


# v0.1.0-beta.6

## New Features

- Fully-featured playbooks with the added ability into the agent to redirect the
  request to a given URL. (#72)

- Configurable protection behaviour of the agent when blocking a request by
  either customizing the HTTP status code that is used for the blocking HTML
  page, or by redirecting to a given URL instead.  
  Dashboard page: https://my.sqreen.com/application/goto/settings/global#protection-mode

- HTTP response status code monitoring. (#75)  
  Dashboard page: https://my.sqreen.com/application/goto/monitoring
  
- Support for browser security headers protection modules allowing to enable
  various browser security options allowing to restrict modern browsers from
  running into some preventable vulnerabilities:

  - [Content Security Policy][csp] protection module allowing to prevent
    cross-site scripting attacks. (#74)  
    Dashboard page: https://my.sqreen.com/application/goto/modules/csp

  - Security headers protection module allowing to protect against client-side
    vulnerabilities in the browser. (#73)  
    Dashboard page: https://my.sqreen.com/application/goto/modules/headers

## Minor Changes

- Better agent configuration logs clearly stating where does the configuration
  come from (file in search path, enforced file or environment variables),
  along with the possibility to display the full settings using the `debug`
  log-level.


# v0.1.0-beta.5

## New Features

- Middleware functions, called interceptors, for gRPC over HTTP2. More details
  on how to use it at
  https://godoc.org/github.com/sqreen/go-agent/sdk/middleware/sqgrpc. (#23)

- [IP whitelist](https://my.sqreen.com/application/goto/settings/whitelist)
  support to make the agent completely ignore requests whose IP addresses are
  whitelisted. Everything related to Sqreen, including events, will be ignored.
  (#69)

- Agent fail-safe catching errors and panics in order to prevent the host Go
  app to fail. The fail-safe mechanism either tries to restart the agent or
  ultimately stops it. (#67)

## Minor Change

- Internal event batch improvements:
  - Increased batch buffer capacity from 60 to 6000 entries in order to be able
    to handle more events, sent by batches of 60 events per heartbeat.
  - Remove a bookkeeping goroutine and include its logic into the main event
    processing loop.


# v0.1.0-beta.4

This release adds the ability to block IP addresses or users into your Go web
services by adding support for [Security Automation] according to your
[playbooks] and their configured security responses.

Note that redirecting users or IP addresses is not supported yet.

## New Feature

- [Security Automation]:  
  It is now possible to block IP addresses or users. When a [playbook]
  triggers, the agent is notified and gets the batch of security responses.
  They are asynchronously stored into data structures optimized for fast lookup
  and low memory usage. Middleware functions can thus perform fast lookups to
  block requests in a few microseconds in order to exit request handlers as
  fast as possible.

  - Blocking IP addresses:  
    No changes are required to block IP addresses. Our middleware functions
    have been updated to block requests whose IP addresses match a security
    response. The request is aborted with HTTP status code `500` and Sqreen's
    default HTML information page.
  
  - Blocking users:   
    Blocking users is performed by combining SDK methods `Identify()` and
    `MatchSecurityResponse()` in order to firstly associate a user to the
    current request, and secondly to check if it matches a security response.
    When a security response matches, the request handler and any related
    goroutines should be stopped as soon as possible.
    
    Usage example:
    ```go
    uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
    sqUser := sdk.FromContext(ctx).ForUser(uid)
    sqUser.Identify()
    if match, err := sqUser.MatchSecurityResponse(); match {
      // Return now to stop further handling the request and let Sqreen's
      // middleware apply the configured security response and abort the
      // request. The returned error may help aborting from sub-functions by
      // returning it to the callers when the Go error handling pattern is
      // used.
      return err
    }
    ```
    
    We strongly recommend to create a user-authentication middleware function
    in order to seamlessly integrate user-blocking to all your
    user-authenticated endpoints.

## Fix

- Escape the event type name to avoid JSON marshaling error. Note that this
  case could not happen in previous agent versions. (#52)

## Minor Change

- Avoid performing multiple times commands within the same command batch. (51)


# v0.1.0-beta.3

## New Features

- Get the client IP address out of the HAProxy header `X-Unique-Id` using the
  new configuration variable `ip_header_format`. (#41)

- New configuration option `strip_http_referer`/`SQREEN_STRIP_HTTP_REFERER`
  allowing to avoid sending the `Referer` HTTP header to the Sqreen backend when
  it contains sensitive data. (#36)

- Ability to disable/enable the agent through the [dashboard
  settings](https://my.sqreen.com/application/goto/settings/global) using the
  Sqreen status button. (#29)

## Breaking Changes

- Agent internals are now under a private Go package and can no longer be
  imported. Any sub-package under `github.com/sqreen/go-agent/agent` was not
  supposed to be imported and is now private to avoid future confusions. (#27)

## Fixes

- Remove duplicate `User-Agent` entry sent twice in the request record. (#42)

- Fix IPv4 and IPv6 matching against private network definitions. (#38)

- Remove useless empty request records mistakenly created while not carrying
  any SDK observation. (#38)

## Minor Changes

- Better memory management and footprint when the agent is disabled by removing
  globals. This will be also required to be able to cleanly restart the agent by
  self-managing the initializations. (#28)


# v0.1.0-beta.2

## New feature

- Add a new `Identify()` method allowing to explicitly associate a user to the
current request. As soon as we add the support for the security reponses, it
will allow to block users (#26).

# v0.1.0-beta.1

This version is a new major version towards the v0.1.0 as it proposes a new and
stable SDK API, that now will only be updated upon user feedback. So please,
share your impressions with us.

## New Features

- New web framework middleware support:
  - Standard Go's `net/http` package (#21).
  - Echo (#19).

- Multiple custom events can now be easily associated to a user using the
  user-scoped methods under `ForUser()`. For example, to send two custom events
  for a given user, do:

    ```go
    sqUser := sqreen.ForUser(uid)
    sqUser.TrackEvent("my.event.one")
    sqUser.TrackEvent("my.event.two")
    ```

- The configuration file can now be stored into multiple locations, the current
  working directory or the executable one, or enforced using the new
  configuration environment variable `SQREEN_CONFIG_FILE` (#25).

- The custom client IP header configured in `SCREEN_IP_HEADER` is now also sent
  to Sqreen so that it can better understand what IP headers were considered by
  the agent to determine what is the actual client IP address
  (67e2d4cbf9b883e9e91e1a5d9e53348a18c1b900).

## Breaking Changes

- Stable SDK API of "Sqreen for Go":

  - Avoid name conflicts with framework packages by prefixing Sqreen's
    middleware packages with `sq`. For example, `gin` becomes `sqgin` (#17).

  - Cleaner Go documentation now entirely included in the SDK and middleware
    packages Go documentations. So no more need to go inside the agent
    documentation to know more on some SDK methods, it is now all documented
    in the same place, with lot of examples.

  - Clearer SDK API: The flow of security events that can send to Sqreen is
    now well-defined by a tree of SDK methods that can only be used the right
    way. (#18, #24)

     - The SDK handle getter function name is renamed from
       `GetHTTPRequestContext()` into a simpler `FromContext()`.

     - User-related SDK methods are now provided by `ForUser()`, for example:

         ```go
         sqreen.TrackAuth(true, uid)
         ```

       becomes

         ```go
         sqreen.ForUser(uid).TrackAuthSuccess()
         ```


# v0.1.0-alpha.5

## New features

- sdk: user-related security events:
      - ability to associate a user to an event using `WithUserIdentifier()` (#13).
      - track user creation using `TrackSignup()` (#15).
      - track user authentication using `TrackAuth()` (#15).
    
- agent/backend: take into account `{HTTPS,HTTP,NO}_PROXY` environment variables (and their lowercase alternatives) (#14).
    
- agent/backend: share the organization token for all your apps (#12).

## Fixes

- agent/config: avoid conflicts with global viper configs (#16).
- sdk: better documentation with examples.

[Security Automation]: https://docs.sqreen.com/security-automation/introduction/
[playbook]: https://docs.sqreen.com/security-automation/introduction-playbooks/
[playbooks]: https://docs.sqreen.com/security-automation/introduction-playbooks/
[csp]: https://docs.sqreen.com/using-sqreen/automatically-set-content-security-policy/
