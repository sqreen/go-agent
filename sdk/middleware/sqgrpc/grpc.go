// Copyright (c) 2016 - 2019 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

// This package provides gRPC interceptors, which are Sqreen's middleware
// functions for gRPC allowing to monitor and protect the received requests. In
// protection mode, it can block and redirect requests according to their IP
// addresses or identified users using `Identify()` and
// `MatchSecurityResponse()` methods. SDK methods can be called from request
// handlers by using the request event record, which can be accessed using
// `sdk.FromContext()` on the request context.
//
// Usage example:
//
// 	myServer := grpc.NewServer(
// 		grpc.StreamInterceptor(sqgrpc.StreamServerInterceptor()),
// 		grpc.UnaryInterceptor(sqgrpc.UnaryServerInterceptor()),
// 	)
//
// 	// Example of a unary RPC doing a custom event.
// 	func (s *MyService) MyUnaryRPC(ctx context.Context, req *pb.MyRequest) (*pb.MyResponse, error) {
// 		sdk.FromContext(ctx).TrackEvent("my.event")
// 		// ...
// 	}
//
// 	// Example of a streaming RPC identifying a user and checking if it should
// 	// be blocked.
// 	func (s *MyService) MyStreamRPC(req *pb.MyRequest, stream pb.MyService_MyStreamRPCServer) error {
// 		// Example of globally identifying a user and checking if the request
// 		// should be aborted.
// 		uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
// 		sqUser := sdk.FromContext(stream.Ctx).ForUser(uid)
// 		sqUser.Identify() // Globally associate this user to the current request
// 		if _, err := sqUser.MatchSecurityResponse(); err != nil {
// 			// Return this error to stop further handling the request and let
// 			// Sqreen's	middleware apply and abort the request.
// 			return err
// 		}
// 		// ... not blocked ...
// 	}
//
package sqgrpc

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/sqreen/go-agent/sdk/middleware/sqhttp"
	"golang.org/x/xerrors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		sqreened := newRequestFromMD(ctx)
		var res interface{}
		err := sqhttp.MiddlewareWithError(sqhttp.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) (err error) {
			res, err = handler(r.Context(), req)
			return err
		})).ServeHTTP(noopHTTPResponseWriter{}, sqreened)
		if xerrors.Is(err, sqhttp.AbortRequestError{}) {
			return nil, status.Error(codes.Aborted, "aborted by sqreen security action")
		}
		return res, err
	}
}

func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		sqreened := newRequestFromMD(stream.Context())
		err := sqhttp.MiddlewareWithError(sqhttp.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) (err error) {
			stream := grpc_middleware.WrapServerStream(stream)
			stream.WrappedContext = r.Context()
			return handler(srv, stream)
		})).ServeHTTP(noopHTTPResponseWriter{}, sqreened)
		if xerrors.Is(err, sqhttp.AbortRequestError{}) {
			return status.Error(codes.Aborted, "aborted by sqreen security action")
		}
		// Note that we do not control the result's payload here. So users need
		// to use the SDK in order to check the user security response and avoid
		// sending messages. A slower solution would be wrapping the stream's
		// Recv() and Send() methods in order to check for the security response
		// every time a message is received/sent, so that the connection can
		// be aborted.
		return nil
	}
}

// http2Request is a convenience type to get HTTP2 header values from gRPC's
// metadata map. This metadata map contains every HTTP2 header.
type http2Request metautils.NiceMD

func (r http2Request) getMethod() string {
	return metautils.NiceMD(r).Get(":method")
}

func (r http2Request) getURL() *url.URL {
	md := metautils.NiceMD(r)
	return &url.URL{
		Scheme:  md.Get(":scheme"),
		Host:    r.getHost(),
		RawPath: r.getRequestURI(),
	}
}

func (r http2Request) getRequestURI() string {
	return metautils.NiceMD(r).Get(":path")
}

func (r http2Request) getHost() string {
	return metautils.NiceMD(r).Get("host")
}

func (r http2Request) getContentLength() int64 {
	lenStr := metautils.NiceMD(r).Get("content-length")
	i, _ := strconv.ParseInt(lenStr, 10, 64)
	return i
}

func (r http2Request) getHeader() http.Header {
	h := make(http.Header)
	md := metautils.NiceMD(r)
	for k, values := range md {
		for _, v := range values {
			h.Add(k, v)
		}
	}
	return h
}

// newRequestFromMD returns the context including the SDK request record along
// with the SDK request handle. For now, it maps gRPC's metadata to aGo-standard
// HTTP request in order to be compatible with the current API. In the future, a
// better abstraction should allow to not rely only on the standard Go HTTP
// package only.
func newRequestFromMD(ctx context.Context) *http.Request {
	// gRPC stores headers into the metadata object.
	r := http2Request(metautils.ExtractIncoming(ctx))
	p, ok := peer.FromContext(ctx)
	var remoteAddr string
	if ok {
		remoteAddr = p.Addr.String()
	}
	return &http.Request{
		Method:        r.getMethod(),
		URL:           r.getURL(),
		Proto:         "HTTP/2",
		ProtoMajor:    2,
		Header:        r.getHeader(),
		ContentLength: r.getContentLength(),
		Host:          r.getHost(),
		RemoteAddr:    remoteAddr,
		RequestURI:    r.getRequestURI(),
	}
}

// TODO: agent interfaces should not require this hack.

// noopHTTPResponseWriter allows to call the security response handler so that
// it performs its event.
type noopHTTPResponseWriter struct{}

// TODO: agent interfaces should not require this hack.

func (noopHTTPResponseWriter) Header() http.Header         { return nil }
func (noopHTTPResponseWriter) Write(b []byte) (int, error) { return len(b), nil }
func (noopHTTPResponseWriter) WriteHeader(statusCode int)  {}
