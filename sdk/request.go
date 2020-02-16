// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package sdk

// HTTPRequest is a convenience type to hold together the request and its
// request record. Most importantly, it is created by `sdk.NewHTTPRequest()` by
// middleware functions to ensure that the request pointer it contains is the
// one having the context value expected by `sdk.FromContext()`.
//type HTTPRequest struct {
//	request *http.Request
//	record  *HTTPRequestRecord
//}
//
//// NewHTTPRequest returns a new HTTP request handle for the given HTTP request.
//// It is a convenience value making sure the wrapped request has the request
//// record value, which can be retrieved using `sdk.FromContext()` to perform.
//func NewHTTPRequest(req *http.Request) *HTTPRequest {
//	if _, disabled := agent.(disabledAgent); disabled {
//		return &HTTPRequest{
//			request: req,
//			record:  nil,
//		}
//	}
//
//	// Create a pointer to a HTTPRequest.
//	record := new(HTTPRequestRecord)
//	// Store it into the request context.
//	ctx := req.Context()
//	contextKey := HTTPRequestRecordContextKey
//	ctx = context.WithValue(ctx, contextKey, record)
//	// Set the request context with the new one.
//	req = req.WithContext(ctx)
//	rr, req := agent.NewRequestRecord(req)
//	// Set the record pointer value using the new request.
//	*record = HTTPRequestRecord{
//		record: rr,
//	}
//
//	return &HTTPRequest{
//		request: req,
//		record:  record,
//	}
//}
//
//func (r *HTTPRequest) Close() {
//	r.record.Close()
//}
//
//func (r *HTTPRequest) Request() *http.Request {
//	return r.request
//}
//
//func (r *HTTPRequest) Record() *HTTPRequestRecord {
//	return r.record
//}
//
//func (r *HTTPRequest) SecurityResponse() http.Handler {
//	record := r.Record()
//	if record == nil {
//		return nil
//	}
//	return record.record.SecurityResponse()
//}
//
//func (r *HTTPRequest) UserSecurityResponse() http.Handler {
//	record := r.Record()
//	if record == nil {
//		return nil
//	}
//	return record.record.UserSecurityResponse()
//}
