# v0.14.0 - 2 September 2020

## New Feature

- (#142) RASP: add Shellshock protection support. This protection is currently
  attached to `os.StartProcess()` which is the common function of the Go
  standard library to execute a process. This protection can be configured at
  <https://my.sqreen.com/application/goto/modules/rasp/details/shellshock>.

## Fixes

- (#145) In-App WAF: always recover from panics as this in the way the `reflect`
  package handles usage errors.

- (#144) Backend client: avoid dropping HTTP traces in case of `Host` header
  parsing errors.


# v0.13.0 - 24 July 2020

## New Feature

- (#137) RASP: add noSQL Injection protection support for the Go MongoDB driver
  `go.mongodb.org/mongo-driver/mongo`. This protection can be configured at
  <https://my.sqreen.com/application/goto/modules/rasp/details/nosql_injection>.

## Internal Changes

- (#138) Health-check the HTTPS connectivity to the new backend API
  `ingestion.sqreen.com` before using it. Fallback to the usual
  `back.sqreen.com` in case of a connection issue. Therefore, the agent can take
  up to 30 seconds to connect to Sqreen if the health-check timeouts. Please
  make sure to add this new  firewall and proxy configurations. 

- (#136) Add support to attach multiple security protections per hook point.

## Fixes

- (#140) Fix the In-App WAF metadata PII scrubbing to also match substrings.


# v0.12.1 - 13 July 2020

## Fixes

- (d81222d) Add missing request parameters when both JSON values and form values
  were present - only the form values were taken into account.

- (ee22b77) Upgrade to libsqreen v0.7.0:
    - Fix false positives in libinjection SQL heuristics.
    - Fix a false positive in libinjection XSS heuristics.
    - Add support for boolean values.
    - Add support for float values.
    - Fix memory deallocator of scalar values.

- (c425760) Fix data bindings with null values.

## Internal Changes

- (eeb1dca) Avoid copying the metadata returned by the In-App WAF.


# v0.12.0 - 6 July 2020

## New Features

- (#130) In-App WAF protection of the HTTP request body:  
  Since the HTTP request handler needs to explicitly read the request body, and
  ultimately parse it into a Go value, the In-App WAF is now applied to new
  points in the request lifecycle:

    1. Reading the request body is now monitored until EOF is reached, and the
       raw body bytes are made available to the In-App WAF rules via a new
       In-App WAF field `Body`. Note that such In-App WAF rules can be created
       on custom In-App WAF rulesets only.

    1. Parsers can be now protected by the In-App WAF once they have parsed a
       request input into a Go value. The parsed value is made available to the
       In-App WAF rules via the `GET/POST parameters` field. Every
       existing In-App WAF rule using this field therefore applies.  
       This new feature is firstly deployed on Gin's [`ShouldBind()`](https://pkg.go.dev/github.com/gin-gonic/gin?tab=doc#Context.ShouldBind)
       method which is Gin's function to parse HTTP request values. It allows to
       cover every parser Gin provides such as [`BindJSON()`](https://pkg.go.dev/github.com/gin-gonic/gin?tab=doc#Context.BindJSON),
       [`BindXML()`](https://pkg.go.dev/github.com/gin-gonic/gin?tab=doc#Context.BindXML), etc.

       When blocked, the function returns a non-nil [`SqreenError` value](https://godoc.org/github.com/sqreen/go-agent/sdk/types#SqreenError)
       and the caller should immediately return.  
       Read more about the blocking behavior of Sqreen for Go at <https://docs.sqreen.com/go/integration>.

- (#129) Update Sqreen's blocking HTML page with a clearer description of what users getting it should do.

## Fix

- (794d6e2) Allow port numbers in the `X-Forwarded-For` header.


# v0.11.0 - 19 June 2020

## New Features

- (#119) RASP: add Shell Injection protection support. This protection is currently dynamically applied to `os.StartProcess()` which is the only entry point of the Go standard library to execute a process.  This protection can be configured at <https://my.sqreen.com/application/goto/modules/rasp/details/shi>.

- (#119) RASP: add Local File Inclusion protection support. This protection is currently dynamically applied to `os.Open()` which is the only entry point of the Go standard library to open a file for reading. This protection can be configured at <https://my.sqreen.com/application/goto/modules/rasp/details/lfi>.

- (#120) RASP: add Server-Side Request Forgery protection support. This protection is currently dynamically applied to `net/http.(*Client).do()` which is the only entry point of the Go standard library to perform an HTTP request. This protection can be configured at <https://my.sqreen.com/application/goto/modules/rasp/details/ssrf>.

- (#125) RASP: enable SQL Injection protection for every MySQL, Oracle, SQLite and PostgreSQL drivers listed in the Go language wiki page <https://github.com/golang/go/wiki/SQLDrivers>.

- (#115) RASP: store Sqreen's request protection context into the Goroutine Local Storage (GLS). Therefore, Sqreen can now protect every Go function without requiring the request Go context (eg. both `QueryContext()` and `Query()` can be now protected against SQL injections). For now, this protection context is only available in the goroutine handling the request, and sub-goroutines are not protected. Further support will be added very soon to remove this limitation.

- (#121) Add IP denylist support: block every request performed by an IP address of the denylist. Every usage of whitelist and blacklist in the agent was also removed when possible. The IP denylist can be configured at <https://my.sqreen.com/application/goto/settings/denylist>.

- (#122) Add path passlist support: requests performed on those paths are not monitored nor protected by Sqreen. The Path passlist can be configured at <https://my.sqreen.com/application/goto/settings/passlist>.

- (#123) Export the error type returned by Sqreen protections when blocking in the new SDK package `github.com/sqreen/go-agent/sdk/types` in order to avoid retrying blocked function calls (eg. avoid retrying a blocked SQL query). It must be used along with `errors.As()` to detect such cases. Read more at <https://godoc.org/github.com/sqreen/go-agent/sdk/types>.

- (#124) Allow to "quickly" remove the agent from a program by only removing it from the source code without disabling the program instrumentation. This is made possible by making the instrumentation fully autonomous to avoid compilation errors.

## Fix

- Gin Middleware: fix the HTTP status code monitoring that was possibly changed by Gin after having been already written.

## Internal Changes

- (#126) Cache request value lookups, mainly to accelerate the In-App WAF when lots of rulesets are enabled.

- (#117) Simpler Go vendoring support implementation.

- (#113) Significant JavaScript performance improvements by changing the virtual machine to `github.com/dop251/goja`.

- (#114) Add Goroutine Local Storage (GLS) support through static instrumentation of the Go runtime.


# v0.10.1 - 5 June 2020

## Fix

- (#116) Fix the instrumentation tool ignoring vendored packages, leading to
  missing hook points in the agent.

# v0.10.0 - 20 May 2020

## New Features

- (#109) Make the PII sanitizer configurable with two new configuration entries
  allowing to control the regular expressions used to sanitize everything sent
  to Sqreen. The agent doesn't start in case of an invalid regular expression.
  More details can be found on the [configuration's documentation page](https://docs.sqreen.com/go/configuration/#personally-identifiable-information-scrubbing).

- (#110) The `net/http` middleware now includes URL segments in the request
  parameters to increase the coverage we have on frameworks compatible with it,
  such as `gorilla` or `beego`.

## Internal Changes

- (#107) Backend API: integrate the security signal HTTP API.

## Fixes

- (#108) Update the token validation to correctly handle the new token format.

- (#111) Fix the JSON serialization function of HTTP headers monitored by the
  agent that could fail depending on the header values. Note that the JSON
  serialization of the parent data structure safely catches any JSON injection
  attempt.

## Documentation

- Add quick start examples for common build and deployment environments such as
  docker images, heroku and google app engine.

- Document Heroku installation at <https://docs.sqreen.com/go/installation/heroku>.

- Document Google App Engine installation at <https://docs.sqreen.com/go/installation/google-app-engine>.

- Document Docker image installation at <https://docs.sqreen.com/go/installation/docker>.

- Document PII scrubbing configuration at <https://docs.sqreen.com/go/configuration/#personally-identifiable-information-scrubbing>.

# v0.9.1 - 31 March 2020

## Fixes

- (#99) Fix mistakenly enforced HTTP status code `200` when Sqreen's middleware
  function is not the first in the request handling chain. This issue appeared
  when not adding Sqreen's middleware function as the root HTTP middleware.

- (#100) Fix the monitoring of HTTP response codes mistakenly considered `200`
  when set by the request handlers.

- (#101) Prevent starting the agent when the instrumentation tool and agent
  versions are not the same.

# v0.9.0 - 19 February 2020

This new major version says farewell to the `beta` and adds SQL-injection
run time protection thanks the first building blocks of [RASP][RASP-Wikipedia]
for the Go language! Thank you to everyone that helped us in this wonderful
and amazing journey ;-)

The Go agent has been protecting production servers for more than a year now and
we reached the point where we are confident enough about its deployment, setup,
but also its internals and specific integrations with the Go language and
runtime.

We are getting closer to the fully-featured agent v1.0 as we will now be able to
fully add support for every RASP protection Sqreen supports.

## Breaking Changes

Because we now want a stable public API, find below the breaking changes:

- The former separate agent package `github.com/sqreen/go-agent/agent` that was
  asked to import in order to start the agent is no longer required nor
  available. This is now performed by the middleware functions we
  provide in order to avoid the most common setup mistake during the
  beta where only the agent was setup and no middleware function was set to
  protect the requests (and therefore nothing was happening).

- SDK: the user identification SDK method `Identify()` has been updated to be
  simpler to use and less error-prone by now making it return a non-nil error
  when the request handler shouldn't continue any further serving the request.
  It happens when a user security response has matched the identified user.
  This replaces the former separate SDK method `MatchSecurityResponse()`.
  New usage example:
  ```go
  sqUser := sq.ForUser(sdk.EventUserIdentifiersMap{"uid": "unique user id"})
  if err := sqUser.Identify(); err != nil {
    return
  }
  ```

- The agent no longer starts if the program wasn't instrumented using the
  instrumentation tool. See docs.sqreen.com/go/installation for details
  on how to install and use the tool. Note that the program is not aborted -
  only the agent is disabled.

- Dropping gRPC support: the beta support for gRPC was experimental and was in
  the end too limited by Sqreen's focus on the HTTP protocol. Most of our
  protections are indeed designed for HTTP and couldn't be applied at the gRPC
  protocol level. We are therefore removing it until we can provide a correct
  experience for such HTTP-based protocol.  
  Please contact us if you need any further information or if you are
  interested in helping us building it (support@sqreen.com).

## New Features

- SQL-injection RASP protection: when enabled on [Sqreen's dashboard](https://my.sqreen.com/application/goto/modules/rasp),
  the `database/sql` Go package gets automatically protected against SQL
  injections. SQL queries go through our SQL-injection detection which will
  abort the SQL function call and corresponding HTTP request when an attack
  is detected.  
  Note that special care was taken to properly intergrate with Go error-handling
  principles: when a SQL query gets blocked, the HTTP request context is
  canceled and a non-nil error is returned by the `database/sql` function call
  in order to fall into the existing error-handling flow. For example:
  ```go
  // The following query can be injected. An error is returned when the SQL
  // query was blocked.
  rows, err := db.QueryContext(ctx, "select id, name from users where id=" + unsafe)
  if err != nil {
    return err
  }
  ```
  Read more about Go integration details at http://docs.sqreen.com/go/integration.

- Dashboard diagnostic messages: major setup issues are now also reported
  through Sqreen's dashboard page of [running hosts](https://my.sqreen.com/application/goto/settings/hosts)
  to get notified about some downgraded states of the agent, such as:
  - The Go program is not instrumented so the agent didn't start.
  - The In-App WAF wasn't compiled (eg. CGO disabled) so it is unavailable and
    disabled.
  - The program dependencies couldn't be retrieved because the program was not
    compiled as a Go module. This is also shown by the dashboard when the list
    of dependencies is empty.

# v0.1.0-beta.10 - 24 January 2020

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

# v0.1.0-beta.9 - 19 December 2019

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

# v0.1.0-beta.8 - 15 October 2019

## Internal Changes

- In-App WAF:
  - Dynamically set the WAF timeout (#79).
  - Ignore WAF timeout errors and add more context when reporting an error (#80).
  - Update the libsqreen to v0.4.0 to add support for the `@pm` operator.

# v0.1.0-beta.7 - 26 September 2019

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


# v0.1.0-beta.6 - 25 July 2019

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


# v0.1.0-beta.5 - 23 May 2019

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


# v0.1.0-beta.4 - 16 April 2019

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


# v0.1.0-beta.3 - 22 March 2019

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


# v0.1.0-beta.2 - 14 February 2019

## New feature

- Add a new `Identify()` method allowing to explicitly associate a user to the
current request. As soon as we add the support for the security reponses, it
will allow to block users (#26).

# v0.1.0-beta.1 - 7 February 2019

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
[RASP-Wikipedia]: https://en.wikipedia.org/wiki/Runtime_application_self-protection
