package sqgrpc_test

import (
	"context"
	"testing"

	grpc_testing "github.com/grpc-ecosystem/go-grpc-middleware/testing"
	pb_testproto "github.com/grpc-ecosystem/go-grpc-middleware/testing/testproto"
	"github.com/sqreen/go-agent/sdk"
	"github.com/sqreen/go-agent/sdk/middleware/sqgrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
)

func assertSqreenHandleExists(t *testing.T, ctx context.Context) {
	require.NotNil(t, sdk.FromContext(ctx), "The middleware should attach its handle object to the request context")
}

type assertingPingService struct {
	pb_testproto.TestServiceServer
	T *testing.T
}

func (s *assertingPingService) Ping(ctx context.Context, ping *pb_testproto.PingRequest) (*pb_testproto.PingResponse, error) {
	assertSqreenHandleExists(s.T, ctx)
	return s.TestServiceServer.Ping(ctx, ping)
}

func (s *assertingPingService) PingList(ping *pb_testproto.PingRequest, stream pb_testproto.TestService_PingListServer) error {
	assertSqreenHandleExists(s.T, stream.Context())
	return s.TestServiceServer.PingList(ping, stream)
}

func TestMiddlewareSuite(t *testing.T) {
	s := &MiddlewareTestSuite{
		InterceptorTestSuite: &grpc_testing.InterceptorTestSuite{
			TestService: &assertingPingService{&grpc_testing.TestPingService{T: t}, t},
			ServerOpts: []grpc.ServerOption{
				grpc.StreamInterceptor(sqgrpc.StreamServerInterceptor()),
				grpc.UnaryInterceptor(sqgrpc.UnaryServerInterceptor()),
			},
		},
	}
	suite.Run(t, s)
}

type MiddlewareTestSuite struct {
	*grpc_testing.InterceptorTestSuite
}

func (s *MiddlewareTestSuite) TestUnary() {
	_, err := s.Client.Ping(s.SimpleCtx(), &pb_testproto.PingRequest{Value: "something", SleepTimeMs: 9999})
	assert.NoError(s.T(), err)
}

func (s *MiddlewareTestSuite) TestStream() {
	stream, err := s.Client.PingList(s.SimpleCtx(), &pb_testproto.PingRequest{Value: "something", SleepTimeMs: 9999})
	assert.NoError(s.T(), err)
	_, err = stream.Recv()
	assert.NoError(s.T(), err)
}
