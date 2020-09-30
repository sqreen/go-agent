<p align="center">
<img width="20%" src="/doc/images/sqreen-gopher.png" alt="Sqreen for Go" title="Sqreen for Go" />
</p>

# [Sqreen](https://www.sqreen.com/)'s Application Security Management for Go

After performance monitoring (APM), error and log monitoring it’s time to add a
security component into your app. Sqreen’s microagent automatically monitors
sensitive app’s routines, blocks attacks and reports actionable infos to your
dashboard.

<p align="center">
<img width="80%" src="https://sqreen-assets.s3-eu-west-1.amazonaws.com/miscellaneous/dashboard.gif" alt="Sqreen for Go" title="Sqreen for Go" />
</p>

# Gin middleware function

This package provides Sqreen's middleware function for Gin to monitor and
protect requests Gin receives. Simply setup the middleware function to have your
request monitored and protected by Sqreen.

Usage:

```go
router := gin.Default()
// Setup Sqreen's middleware
router.Use(sqgin.Middleware())

// Every router endpoint is now automatically monitored and protected by Sqreen
router.GET("/", func(c *gin.Context) {
  c.Status(http.StatusOK)
}
```

Find more details on how to setup Sqreen for Go at
<https://docs.sqreen.com/go/installation/>