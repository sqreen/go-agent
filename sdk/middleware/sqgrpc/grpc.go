// Package sqgrpc provides gRPC interceptors. The implementation is in early
// stage with some limitations and for now expects gRPC over HTTP2, as specified
// in https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md.

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

// UnaryServerInterceptor is Sqreen's middleware function for unary RPCs to
// monitor and protect the requests gRPC receives. It creates and stores the
// HTTP2 request record both into the gRPC context so that it can be later
// accessed from handlers using `sdk.FromContext()` to perform SDK calls.
//
// Simple add Sqreen's interceptors to your gRPC server options:
//
//  myServer := grpc.NewServer(
//  	grpc.StreamInterceptor(sqgrpc.StreamServerInterceptor()),
//  	grpc.UnaryInterceptor(sqgrpc.UnaryServerInterceptor()),
//  )
//
// And access the SDK from the RPC endpoints:
//
//	func (s *MyService) MyUnaryRPC(ctx context.Context, req *pb.MyRequest) (*pb.MyResponse, error) {
//		sdk.FromContext(ctx).TrackEvent("my.event")
//		// ...
//	}
//
//	func (s *MyService) MyStreamRPC(req *pb.MyRequest, stream pb.MyService_MyStreamRPCServer) error {
//		sdk.FromContext(stream.ctx).TrackEvent("my.event")
//		// ...
//	}
//
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

type noopHTTPResponseWriter struct{}

func (noopHTTPResponseWriter) Header() http.Header         { return nil }
func (noopHTTPResponseWriter) Write(b []byte) (int, error) { return len(b), nil }
func (noopHTTPResponseWriter) WriteHeader(statusCode int)  {}
