package sdk

// HTTPRequestRecordContextKey is a context key. It can be used in HTTP
// handlers with context.Context.Value() to access the HTTPRequestRecord that
// was associated with the request by the middleware. The associated value will
// be of type *HTTPRequestRecord.
var HTTPRequestRecordContextKey = &ContextKey{"sqreen.rr"}

// ContextKey allows to insert context values with comparable keys that are not
// strings, as documented by context.WithValue(), to avoid string collisions.
type ContextKey struct {
	// This string value must be used by middelware function whose framework
	// expects context keys of type string, such as Gin.
	String string
}
