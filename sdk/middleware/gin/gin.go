package gin

import (
	"context"
	"net/http"

	gingonic "github.com/gin-gonic/gin"
	sqreen_sdk "github.com/sqreen/go-agent/sdk"
)

const agentCtxGinKey = "sq.ctx"

// Sqreen is the middleware function for Gin so that it monitors and protects
// received requests. It creates and stores the sdk's context into the Gin's and
// request's contextes so that they can be retrieved from handlers to perform
// sdk calls using GetHTTPContextFromGin() or GetHTTPContext().
func Sqreen() gingonic.HandlerFunc {
	return func(c *gingonic.Context) {
		// Create a sqreen context for this request.
		sqreen := sqreen_sdk.NewHTTPRequestContext(request{c.Copy()})

		// Store it into Go's context.
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, agentCtxGinKey, sqreen)
		c.Request = c.Request.WithContext(ctx)

		// Store it into Gin's context.
		c.Set(agentCtxGinKey, sqreen)

		c.Next()

		// Close the sqreen context
		sqreen.Close()
	}
}

// GetHTTPContextFromGin returns the sdk's context associated to the Gin's
// context by the middleware function.
//
//	router.GET("/", func(c *gin.Context) {
//		sqreen_middleware.GetHTTPContextFromGin(c).Track("my.event")
//		// ...
//	}
//
func GetHTTPContextFromGin(c *gingonic.Context) *sqreen_sdk.HTTPRequestContext {
	ctx, exists := c.Get(agentCtxGinKey)
	if !exists {
		return nil
	}
	return ctx.(*sqreen_sdk.HTTPRequestContext)
}

// GetHTTPContext returns the sdk's context associated to the request's context by the
// middleware function.
//
// It allows to retrieve it out of a Go context when the Gin context is not
// accessible in the current function scope. So instead of using
// GetHTTPContextFromGin(), you can use this function:
//
//	router.GET("/", func(c *gin.Context) {
//		aFunction(c.Request.Context())
//	}
//
//	func aFunction(ctx context.Context) {
//		sqreen_middleware.GetHTTPContext(ctx).Track("my.event")
//		// ...
//	}
//
func GetHTTPContext(ctx context.Context) *sqreen_sdk.HTTPRequestContext {
	sqreen := ctx.Value(agentCtxGinKey)
	if sqreen == nil {
		return nil
	}
	return sqreen.(*sqreen_sdk.HTTPRequestContext)
}

type request struct {
	*gingonic.Context
}

func (r request) StdRequest() *http.Request {
	return r.Context.Request
}
