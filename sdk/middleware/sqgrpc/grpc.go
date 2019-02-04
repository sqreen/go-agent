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
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
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
		// Create a new request record
		sqreen := sdk.NewHTTPRequestRecord(requestFromMD(ctx))
		defer sqreen.Close()

		// Store it into the context.
		contextKey := sdk.HTTPRequestRecordContextKey
		newCtx := context.WithValue(ctx, contextKey, sqreen)
		return handler(newCtx, req)
	}
}

func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		sqreen := sdk.NewHTTPRequestRecord(requestFromMD(ctx))
		defer sqreen.Close()

		// Store it into the stream context.
		contextKey := sdk.HTTPRequestRecordContextKey
		newCtx := context.WithValue(ctx, contextKey, sqreen)
		wrapped := grpc_middleware.WrapServerStream(stream)
		wrapped.WrappedContext = newCtx
		return handler(srv, wrapped)
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

func requestFromMD(ctx context.Context) *http.Request {
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
