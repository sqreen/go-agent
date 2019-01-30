package sdk

// HTTPRequestRecordContextKey is a context key. It can be used in HTTP
// handlers with context.Context.Value() to access the HTTPRequestRecord that
// was associated with the request by the middleware. The associated value will
// be of type *HTTPRequestRecord.
var HTTPRequestRecordContextKey = &ContextKey{"sqreen.rr"}

// ContextKey allows to insert context values avoiding string collisions. Cf.
// `context.WithValue()`.
type ContextKey struct {
	// This string value must be used by middelware functions whose framework
	// expects context keys of type string, such as Gin. `sdk.FromContext()`
	// expect this behaviour to fallback to string keys when getting the value
	// from the pointer address returned null.
	String string
}
