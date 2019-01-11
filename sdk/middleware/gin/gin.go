package gin

import (
	"net/http"

	gingonic "github.com/gin-gonic/gin"
	sqreen_sdk "github.com/sqreen/go-agent/sdk"
)

const agentCtxGinKey = "sq.ctx"

type request struct {
	*gingonic.Context
}

func (r request) StdRequest() *http.Request {
	return r.Context.Request
}

func Sqreen() gingonic.HandlerFunc {
	return func(c *gingonic.Context) {
		sqreen := sqreen_sdk.NewHTTPRequestContext(request{c.Copy()})
		c.Set(agentCtxGinKey, sqreen)
		c.Next()
		sqreen.Close()
	}
}

func GetHTTPContext(c *gingonic.Context) *sqreen_sdk.HTTPRequestContext {
	ctx, exists := c.Get(agentCtxGinKey)
	if !exists {
		return nil
	}
	return ctx.(*sqreen_sdk.HTTPRequestContext)
}
