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
