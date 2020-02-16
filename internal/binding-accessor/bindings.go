// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package bindingaccessor

// BindingAccessorContext is wrapper type of a request providing
// the binding accessor interface expected by rules:
//   - `#.Request` returns the request binding accessor context
//type BindingAccessorContext struct {
//	Request *HTTPRequestBindingAccessorContext
//}
//
//func MakeBindingAccessorContext(req *http.Request, clientIP string) BindingAccessorContext {
//	return BindingAccessorContext{
//		Request: NewHTTPRequestBindingAccessorContext(req, clientIP),
//	}
//}
