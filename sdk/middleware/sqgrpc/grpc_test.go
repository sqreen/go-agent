package sqgrpc_test

import (
	"context"
	"net/http"
	"testing"

	grpc_testing "github.com/grpc-ecosystem/go-grpc-middleware/testing"
	pb_testproto "github.com/grpc-ecosystem/go-grpc-middleware/testing/testproto"
	"github.com/sqreen/go-agent/sdk"
	"github.com/sqreen/go-agent/sdk/middleware/sqgrpc"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func assertSqreenHandleExists(ctx context.Context) {
	if sdk.FromContext(ctx) == nil {
		panic("The middleware should attach its handle object to the request context")
	}
}

type assertingPingService struct {
	pb_testproto.TestServiceServer
}

func (s *assertingPingService) Ping(ctx context.Context, ping *pb_testproto.PingRequest) (*pb_testproto.PingResponse, error) {
	assertSqreenHandleExists(ctx)
	return s.TestServiceServer.Ping(ctx, ping)
}

func (s *assertingPingService) PingList(ping *pb_testproto.PingRequest, stream pb_testproto.TestService_PingListServer) error {
	ctx := stream.Context()
	assertSqreenHandleExists(ctx)
	if ping.ErrorCodeReturned == 42 {
		// Special value to check user blocking
		sqreen := sdk.FromContext(ctx)
		sqUser := sqreen.ForUser(sdk.EventUserIdentifiersMap{})
		sqUser.Identify()
		if _, err := sqUser.MatchSecurityResponse(); err != nil {
			return err
		}
	}
	return s.TestServiceServer.PingList(ping, stream)
}

func TestMiddleware(t *testing.T) {
	t.Run("with sqreen", func(t *testing.T) {
		s := &MiddlewareTestSuite{
			InterceptorTestSuite: &grpc_testing.InterceptorTestSuite{
				TestService: &assertingPingService{&grpc_testing.TestPingService{}},
				ServerOpts: []grpc.ServerOption{
					grpc.StreamInterceptor(sqgrpc.StreamServerInterceptor()),
					grpc.UnaryInterceptor(sqgrpc.UnaryServerInterceptor()),
				},
			},
		}
		suite.Run(t, s)
	})
}

//
//func TestMiddleware(t *testing.T) {
//	s := &grpc_testing.InterceptorTestSuite{
//		TestService: &s.TassertingPingService{&grpc_testing.TestPingService{T: t}, t},
//		ServerOpts: []grpc.ServerOption{
//			grpc.StreamInterceptor(sqgrpc.StreamServerInterceptor()),
//			grpc.UnaryInterceptor(sqgrpc.UnaryServerInterceptor()),
//		},
//	}
//	s.SetT(t)
//	s.SetupSuite()
//	defer s.TearDownSuite()
//	agent, _ := testlib.NewAgentForMiddlewareTestsWithoutSecurityResponse()
//	sdk.SetAgent(agent)
//	_, err := s.Client.Ping(s.SimpleCtx(), &pb_testproto.PingRequest{Value: "something", SleepTimeMs: 9999})
//	assert.NoError(s.T(), err)
//}

type MiddlewareTestSuite struct {
	*grpc_testing.InterceptorTestSuite
}

func (s *MiddlewareTestSuite) TestUnary() {
	t := s.T()
	t.Run("without security response", func(t *testing.T) {
		agent, record := testlib.NewAgentForMiddlewareTestsWithoutSecurityResponse()
		sdk.SetAgent(agent)
		defer agent.AssertExpectations(t)
		defer record.AssertExpectations(t)
		_, err := s.Client.Ping(s.SimpleCtx(), &pb_testproto.PingRequest{Value: "something", SleepTimeMs: 9999})
		require.NoError(t, err)
	})

	t.Run("with security response", func(t *testing.T) {
		t.Run("with ip response", func(t *testing.T) {
			agent, record := testlib.NewAgentForMiddlewareTestsWithSecurityResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(500)
			}))
			sdk.SetAgent(agent)
			defer agent.AssertExpectations(t)
			defer record.AssertExpectations(t)
			_, err := s.Client.Ping(s.SimpleCtx(), &pb_testproto.PingRequest{Value: "something", SleepTimeMs: 9999})
			require.Error(t, err)
			require.Equal(t, status.Code(err), codes.Aborted)
		})

		t.Run("with user response", func(t *testing.T) {
			agent, record := testlib.NewAgentForMiddlewareTestsWithUserSecurityResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(500)
			}))
			sdk.SetAgent(agent)
			defer agent.AssertExpectations(t)
			defer record.AssertExpectations(t)
			_, err := s.Client.Ping(s.SimpleCtx(), &pb_testproto.PingRequest{Value: "something", SleepTimeMs: 9999})
			require.Error(t, err)
			require.Equal(t, status.Code(err), codes.Aborted)
		})
	})
}

func (s *MiddlewareTestSuite) TestStream() {
	t := s.T()
	t.Run("without security response", func(t *testing.T) {
		agent, record := testlib.NewAgentForMiddlewareTestsWithoutSecurityResponse()
		sdk.SetAgent(agent)
		defer agent.AssertExpectations(t)
		defer record.AssertExpectations(t)

		stream, err := s.Client.PingList(s.SimpleCtx(), &pb_testproto.PingRequest{Value: "something", SleepTimeMs: 9999})
		require.NoError(t, err)
		_, err = stream.Recv()
		require.NoError(t, err)
	})

	t.Run("with security response", func(t *testing.T) {
		t.Run("with ip response", func(t *testing.T) {
			agent, record := testlib.NewAgentForMiddlewareTestsWithSecurityResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(500)
			}))
			sdk.SetAgent(agent)
			defer agent.AssertExpectations(t)
			defer record.AssertExpectations(t)

			stream, err := s.Client.PingList(s.SimpleCtx(), &pb_testproto.PingRequest{Value: "something", SleepTimeMs: 9999})
			require.NoError(t, err)
			res, err := stream.Recv()
			require.Error(t, err)
			require.Equal(t, status.Code(err), codes.Aborted)
			require.Nil(t, res)
		})

		t.Run("with user response", func(t *testing.T) {
			agent, record := testlib.NewAgentForMiddlewareTestsWithUserSecurityResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(500)
			}))
			sdk.SetAgent(agent)
			defer agent.AssertExpectations(t)
			record.ExpectIdentify(sdk.EventUserIdentifiersMap{})
			defer record.AssertExpectations(t)

			stream, err := s.Client.PingList(context.TODO(), &pb_testproto.PingRequest{ErrorCodeReturned: 42, Value: "something", SleepTimeMs: 9999})
			require.NoError(t, err)
			res, err := stream.Recv()
			require.Error(t, err)
			require.Equal(t, status.Code(err), codes.Aborted)
			require.Nil(t, res)
		})
	})
}
