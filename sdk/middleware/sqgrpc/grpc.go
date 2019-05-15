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
//		// Add the interceptor options to your gRPC server.
// 		myServer := grpc.NewServer(
//			grpc.StreamInterceptor(sqgrpc.StreamServerInterceptor()),
//  		grpc.UnaryInterceptor(sqgrpc.UnaryServerInterceptor()),
//  	)
//
// 		// Example of a unary RPC doing a custom event.
// 		func (s *MyService) MyUnaryRPC(ctx context.Context, req *pb.MyRequest) (*pb.MyResponse, error) {
//			sdk.FromContext(ctx).TrackEvent("my.event")
//			// ...
//		}
//
// 		// Example of a streaming RPC identifying a user and checking if it should
// 		// be blocked.
// 		func (s *MyService) MyStreamRPC(req *pb.MyRequest, stream pb.MyService_MyStreamRPCServer) error {
//			// Example of globally identifying a user and checking if the request
//			// should be aborted.
//			uid := sdk.EventUserIdentifiersMap{"uid": "my-uid"}
//			sqUser := sdk.FromContext(stream.Ctx).ForUser(uid)
//			sqUser.Identify() // Globally associate this user to the current request
//			if _, err := sqUser.MatchSecurityResponse(); err != nil {
//				// Return this error to stop further handling the request and let
//				// Sqreen's	middleware apply and abort the request.
//				return err
//			}
//			// ... not blocked ...
//		}
//
package sqgrpc

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/sqreen/go-agent/sdk"
	"golang.org/x/xerrors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Create a new sqreen request wrapper.
		ctx, sqreened := newRequestFromMD(ctx)
		defer sqreened.Close()

		// Check if an early security action is already required such as based on
		// the request IP address.
		if handler := sqreened.SecurityResponse(); handler != nil {
			// TODO: better interface for non-standard HTTP packages to avoid this
			//  noopHTTPResponseWriter hack just to send the block event.
			handler.ServeHTTP(noopHTTPResponseWriter{}, sqreened.Request())
			return nil, status.Error(codes.Aborted, "aborted by sqreen security action")
		}

		res, err := handler(ctx, req)
		if err != nil && !xerrors.As(err, &sdk.SecurityResponseMatch{}) {
			// The error is not a security response match
			return res, err
		}

		// Check if a security response should be applied now after having used
		// `Identify()` and `MatchSecurityResponse()`.
		if handler := sqreened.UserSecurityResponse(); handler != nil {
			// TODO: same as before
			handler.ServeHTTP(noopHTTPResponseWriter{}, sqreened.Request())
			return nil, status.Error(codes.Aborted, "aborted by a sqreen user action")
		}

		return res, nil
	}
}

func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx, sqreened := newRequestFromMD(stream.Context())
		defer sqreened.Close()

		// Check if an early security action is already required such as based on
		// the request IP address.
		if handler := sqreened.SecurityResponse(); handler != nil {
			// TODO: better interface for non-standard HTTP packages to avoid this
			//  noopHTTPResponseWriter hack just to send the block event.
			handler.ServeHTTP(noopHTTPResponseWriter{}, sqreened.Request())
			return status.Error(codes.Aborted, "aborted by a sqreen security action")
		}

		wrapped := grpc_middleware.WrapServerStream(stream)
		wrapped.WrappedContext = ctx
		err := handler(srv, wrapped)
		if err != nil && !xerrors.As(err, &sdk.SecurityResponseMatch{}) {
			// The error is not a security response match
			return err
		}

		// Check if a security response should be applied now after having used
		// `Identify()` and `MatchSecurityResponse()`.
		if handler := sqreened.UserSecurityResponse(); handler != nil {
			// TODO: same as before
			handler.ServeHTTP(noopHTTPResponseWriter{}, sqreened.Request())
			return status.Error(codes.Aborted, "aborted by a sqreen user action")
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
// with the SDK request handle. For now, it maps gRPC's metdata to a Go-standard
// HTTP request in order to be compatible with the current API. In the future,
// a better abstraction should allow to not rely only on the standard Go
// HTTP package only.
func newRequestFromMD(ctx context.Context) (context.Context, *sdk.HTTPRequest) {
	// gRPC stores headers into the metadata object.
	r := http2Request(metautils.ExtractIncoming(ctx))
	p, ok := peer.FromContext(ctx)
	var remoteAddr string
	if ok {
		remoteAddr = p.Addr.String()
	}
	req := &http.Request{
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
	req = req.WithContext(ctx)

	// Create a new sqreened request.
	sqreened := sdk.NewHTTPRequest(req)
	// Get the new request context which includes the request record pointer.
	ctx = sqreened.Request().Context()

	return ctx, sqreened
}

// TODO: agent interfaces should not require this hack.

// noopHTTPResponseWriter allows to call the security response handler so that
// it performs its event.
type noopHTTPResponseWriter struct{}

// TODO: agent interfaces should not require this hack.

func (noopHTTPResponseWriter) Header() http.Header         { return nil }
func (noopHTTPResponseWriter) Write(b []byte) (int, error) { return len(b), nil }
func (noopHTTPResponseWriter) WriteHeader(statusCode int)  {}
