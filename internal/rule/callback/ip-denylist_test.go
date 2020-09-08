// Copyright (c) 2016 - 2020 Sqreen. All Rights Reserved.
// Please refer to our terms for more information:
// https://www.sqreen.io/terms.html

package callback_test

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/sqreen/go-agent/internal/actor"
	"github.com/sqreen/go-agent/internal/plog"
	protectioncontext "github.com/sqreen/go-agent/internal/protection/context"
	httpprotection "github.com/sqreen/go-agent/internal/protection/http"
	"github.com/sqreen/go-agent/internal/protection/http/types"
	"github.com/sqreen/go-agent/internal/rule/callback"
	sdktypes "github.com/sqreen/go-agent/sdk/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	errors "golang.org/x/xerrors"
)

type AgentMock struct {
	mock.Mock
}

func (a *AgentMock) FindActionByIP(ip net.IP) (action actor.Action, exists bool, err error) {
	ret := a.Called(ip)
	action, _ = ret.Get(0).(actor.Action)
	exists = ret.Bool(1)
	err = ret.Error(2)
	return
}

func (a *AgentMock) FindActionByUserID(userID map[string]string) (action actor.Action, exists bool) {
	ret := a.Called(userID)
	action, _ = ret.Get(0).(actor.Action)
	exists = ret.Bool(1)
	return
}

func (a *AgentMock) Logger() (logger *plog.Logger) {
	logger, _ = a.Called().Get(0).(*plog.Logger)
	return
}

func (a *AgentMock) Config() (cfg protectioncontext.ConfigReader) {
	cfg, _ = a.Called().Get(0).(protectioncontext.ConfigReader)
	return
}

func (a *AgentMock) SendClosedRequestContext(face protectioncontext.ClosedRequestContextFace) error {
	return a.Called(face).Error(0)
}

func (a *AgentMock) IsIPAllowed(ip net.IP) bool {
	return a.Called(ip).Bool(0)
}

func (a *AgentMock) IsPathAllowed(path string) bool {
	return a.Called(path).Bool(0)
}

func (a *AgentMock) ExpectLogger() *mock.Call {
	return a.On("Logger")
}

type RequestReaderMock struct {
	mock.Mock
}

func (r *RequestReaderMock) Header(header string) (value *string) {
	value, _ = r.Called(header).Get(0).(*string)
	return
}

func (r *RequestReaderMock) Headers() (headers http.Header) {
	headers, _ = r.Called().Get(0).(http.Header)
	return
}

func (r *RequestReaderMock) Method() string {
	return r.Called().String(0)
}

func (r *RequestReaderMock) URL() (u *url.URL) {
	u, _ = r.Called().Get(0).(*url.URL)
	return
}

func (r *RequestReaderMock) RequestURI() string {
	return r.Called().String(0)
}

func (r *RequestReaderMock) Host() string {
	return r.Called().String(0)
}

func (r *RequestReaderMock) RemoteAddr() string {
	return r.Called().String(0)
}

func (r *RequestReaderMock) IsTLS() bool {
	return r.Called().Bool(0)
}

func (r *RequestReaderMock) UserAgent() string {
	return r.Called().String(0)
}

func (r *RequestReaderMock) Referer() string {
	return r.Called().String(0)
}

func (r *RequestReaderMock) QueryForm() (v url.Values) {
	v, _ = r.Called().Get(0).(url.Values)
	return
}

func (r *RequestReaderMock) PostForm() (v url.Values) {
	v, _ = r.Called().Get(0).(url.Values)
	return
}

func (r *RequestReaderMock) ClientIP() (ip net.IP) {
	ip, _ = r.Called().Get(0).(net.IP)
	return
}

func (r *RequestReaderMock) ExpectClientIP() *mock.Call {
	return r.On("ClientIP")
}

func (r *RequestReaderMock) Params() (v types.RequestParamMap) {
	v, _ = r.Called().Get(0).(types.RequestParamMap)
	return
}

func (r *RequestReaderMock) Body() (body []byte) {
	body, _ = r.Called().Get(0).([]byte)
	return
}

type ResponseWriterMock struct {
	mock.Mock
}

func (r *ResponseWriterMock) Header() (h http.Header) {
	h, _ = r.Called().Get(0).(http.Header)
	return
}

func (r *ResponseWriterMock) Write(bytes []byte) (int, error) {
	ret := r.Called(bytes)
	return ret.Int(0), ret.Error(0)
}

func (r *ResponseWriterMock) WriteHeader(statusCode int) {
	r.Called(statusCode)
}

func (r *ResponseWriterMock) WriteString(s string) (n int, err error) {
	ret := r.Called(s)
	return ret.Int(0), ret.Error(1)
}

var logger = plog.NewLogger(plog.Debug, os.Stderr, 0)

func TestIPDenyListCallback(t *testing.T) {
	t.Run("Constructor", func(t *testing.T) {
		t.Run("Configuration errors", func(t *testing.T) {
			for _, tc := range []interface{}{
				nil,
				33,                             // wrong type
				([]interface{})(nil),           // empty list
				[]interface{}{},                // empty list
				[]interface{}{([]string)(nil)}, // empty list
				[]interface{}{[]string{}},      // empty list
			} {
				tc := tc
				t.Run("", func(t *testing.T) {
					r := &RuleContextMockup{}
					defer r.AssertExpectations(t)

					cfg := &NativeCallbackConfigMockup{}
					cfg.ExpectData().Return(tc)
					defer cfg.AssertExpectations(t)

					_, err := callback.NewIPDenyListCallback(r, cfg)
					require.Error(t, err)
				})
			}
		})
	})

	t.Run("Callback", func(t *testing.T) {
		r := &RuleContextMockup{}

		cfg := &NativeCallbackConfigMockup{}
		// Note that exhaustive tests of the underlying IP list is done in the
		// corresponding package, and that we are only testing the callback API here
		// with a few examples.
		data := []interface{}{
			[]string{
				"1.2.3.4",
				"10.0.0.0/8",
			},
		}
		cfg.ExpectData().Return(data).Once()
		defer cfg.AssertExpectations(t)

		cb, err := callback.NewIPDenyListCallback(r, cfg)
		require.NoError(t, err)
		require.NotNil(t, cb)

		prolog, ok := cb.(callback.IPDenyListPrologCallbackType)
		require.True(t, ok)

		ctx, agent, requestReader, _ := newMockups()
		agent.ExpectLogger().Return(logger)

		// Not blocked
		requestReader.ExpectClientIP().Return(net.ParseIP("11.22.33.44")).Once()
		epilog, err := prolog(&ctx)
		require.Nil(t, epilog)
		require.NoError(t, err)
		requestReader.AssertExpectations(t)
		// Make sure we didn't call PushMetricsValue
		r.AssertExpectations(t)

		// Blocked
		ipStr := "1.2.3.4"
		ip := net.ParseIP(ipStr)
		requestReader.ExpectClientIP().Return(ip).Once()
		r.ExpectPushMetricsValue(ipStr, 1).Return(nil).Once()
		epilog, err = prolog(&ctx)
		require.NotNil(t, epilog)
		require.NoError(t, err)
		requestReader.AssertExpectations(t)
		r.AssertExpectations(t)

		var e error
		epilog(&e)
		require.NoError(t, err)
		require.True(t, errors.As(e, &sdktypes.SqreenError{}))
		var actualErr callback.IPDenyListError
		require.True(t, errors.As(e, &actualErr))
		require.Equal(t, ipStr, actualErr.DenyListEntry)
		require.Equal(t, ip, actualErr.DeniedIP)
	})
}

func newMockups() (*httpprotection.RequestContext, *AgentMock, *RequestReaderMock, *ResponseWriterMock) {
	// TODO: lower-level callback expecting the interface it needs, so that
	//   tests are easier and thus faster to write
	agent := &AgentMock{}
	requestReader := &RequestReaderMock{}
	responseWriter := &ResponseWriterMock{}

	ctx := &httpprotection.RequestContext{
		RequestContext: &protectioncontext.RequestContext{
			AgentFace: agent,
		},
		RequestReader:  requestReader,
		ResponseWriter: responseWriter,
	}
	return ctx, agent, requestReader, responseWriter
}
