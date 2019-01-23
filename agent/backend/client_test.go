package backend_test

import (
	"io/ioutil"
	math_rand "math/rand"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/sqreen/go-agent/agent/backend"
	"github.com/sqreen/go-agent/agent/backend/api"
	"github.com/sqreen/go-agent/agent/config"
	"github.com/sqreen/go-agent/tools/testlib"
	"github.com/stretchr/testify/require"
)

var (
	seed = time.Now().UnixNano()
	popr = math_rand.New(math_rand.NewSource(seed))
)

var _ = Describe("The backend client", func() {
	var (
		server *ghttp.Server
		client *backend.Client
	)

	JustBeforeEach(func() {
		var err error
		server = ghttp.NewServer()
		client, err = backend.NewClient(server.URL())
		Expect(err).NotTo(HaveOccurred())
	})

	JustAfterEach(func() {
		server.Close()
	})

	Describe("request", func() {
		var (
			endpointCfg *config.HTTPAPIEndpoint
			statusCode  = http.StatusOK
			response    proto.Message
			request     proto.Message
			headers     http.Header
		)

		JustBeforeEach(func() {
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest(endpointCfg.Method, endpointCfg.URL),
				ghttp.VerifyHeader(headers),
				VerifyJSONPBRepresenting(request),
				RespondWithJSONPB(&statusCode, response),
			))
		})

		Describe("AppLogin", func() {
			var (
				token   string = "my-token"
				appName string = testlib.RandString(2, 50)
			)

			BeforeEach(func() {
				endpointCfg = &config.BackendHTTPAPIEndpoint.AppLogin

				response = api.NewPopulatedAppLoginResponse(popr, false)
				err := JSONPBLoopback(response)
				Expect(err).ToNot(HaveOccurred())

				request = api.NewPopulatedAppLoginRequest(popr, false)
				err = JSONPBLoopback(request)
				Expect(err).ToNot(HaveOccurred())

				headers = http.Header{
					config.BackendHTTPAPIHeaderToken:   []string{token},
					config.BackendHTTPAPIHeaderAppName: []string{appName},
				}
			})

			It("should perform the API call", func() {
				res, err := client.AppLogin(request.(*api.AppLoginRequest), token, appName)
				Expect(err).NotTo(HaveOccurred())
				Expect(res).Should(Equal(response))
			})
		})

		Describe("AppBeat", func() {
			var session string = "my-session"

			BeforeEach(func() {
				endpointCfg = &config.BackendHTTPAPIEndpoint.AppBeat

				response = api.NewPopulatedAppBeatResponse(popr, false)
				err := JSONPBLoopback(response)
				Expect(err).ToNot(HaveOccurred())

				request = api.NewPopulatedAppBeatRequest(popr, false)
				err = JSONPBLoopback(request)
				Expect(err).ToNot(HaveOccurred())

				headers = http.Header{
					config.BackendHTTPAPIHeaderSession: []string{session},
				}
			})

			It("should perform the API call", func() {
				res, err := client.AppBeat(request.(*api.AppBeatRequest), session)
				Expect(err).NotTo(HaveOccurred())
				Expect(res).Should(Equal(response))
			})
		})

	})
})

// JSONPBLoopback passes msg through the JSON-PB marshaler and unmarshaler so
// that msg then has the same data has another protobuf parsed from a JSONPB
// source. Used by loopback tests of API calls.
func JSONPBLoopback(msg proto.Message) error {
	msgJSON, err := api.DefaultJSONPBMarshaler.MarshalToString(msg)
	if err != nil {
		return err
	}
	msg.Reset()
	return jsonpb.UnmarshalString(msgJSON, msg)
}

func RespondWithJSONPB(statusCode *int, object proto.Message, optionalHeader ...http.Header) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		data, err := api.DefaultJSONPBMarshaler.MarshalToString(object)
		Expect(err).ShouldNot(HaveOccurred())
		var headers http.Header
		if len(optionalHeader) == 1 {
			headers = optionalHeader[0]
		} else {
			headers = make(http.Header)
		}
		if _, found := headers["Content-Type"]; !found {
			headers["Content-Type"] = []string{"application/json"}
		}
		copyHeader(headers, w.Header())
		w.WriteHeader(*statusCode)
		w.Write([]byte(data))
	}
}

func VerifyJSONPBRepresenting(expected proto.Message) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyContentType("application/json"),
		func(w http.ResponseWriter, req *http.Request) {
			body, err := ioutil.ReadAll(req.Body)
			Expect(err).ShouldNot(HaveOccurred())
			req.Body.Close()

			expectedType := reflect.TypeOf(expected)
			actualValuePtr := reflect.New(expectedType.Elem())

			actual, ok := actualValuePtr.Interface().(proto.Message)
			Expect(ok).Should(BeTrue(), "Message value is not a proto.Message")

			err = jsonpb.UnmarshalString(string(body), actual)
			Expect(err).ShouldNot(HaveOccurred(), "Failed to unmarshal protobuf")

			Expect(actual).Should(Equal(expected), "ProtoBuf Mismatch")
		},
	)
}

func copyHeader(src http.Header, dst http.Header) {
	for key, value := range src {
		dst[key] = value
	}
}

func TestProxy(t *testing.T) {
	// ghttp uses gomega global functions so globally register `t` to gomega.
	RegisterTestingT(t)
	t.Run("HTTPS_PROXY", func(t *testing.T) { testProxy(t, "HTTPS_PROXY") })
	t.Run("SQREEN_PROXY", func(t *testing.T) { testProxy(t, "SQREEN_PROXY") })
}

func testProxy(t *testing.T, envVar string) {
	t.Skip()
	// FIXME: (i) use an actual proxy, (ii) check requests go through it, (iii)
	// use a fake backend and check the requests exactly like previous tests
	// (ideally reuse them and add the proxy).
	http.DefaultTransport.(*http.Transport).CloseIdleConnections()
	// Create a fake proxy checking it receives a CONNECT request.
	proxy := ghttp.NewServer()
	defer proxy.Close()
	proxy.AppendHandlers(ghttp.CombineHandlers(
		ghttp.VerifyRequest(http.MethodConnect, ""),
		ghttp.RespondWith(http.StatusOK, nil),
	))

	//back := ghttp.NewUnstartedServer()
	//back.HTTPTestServer.Listener.Close()
	//listener, _ := net.Listen("tcp", testlib.GetNonLoopbackIP().String()+":0")
	//back.HTTPTestServer.Listener = listener
	//back.Start()
	//defer back.Close()
	//back.AppendHandlers(ghttp.CombineHandlers(
	//	ghttp.VerifyRequest(http.MethodPost, "/sqreen/v1/app-login"),
	//	ghttp.RespondWith(http.StatusOK, nil),
	//))

	// Setup the configuration
	os.Setenv(envVar, proxy.URL())
	defer os.Unsetenv(envVar)
	require.Equal(t, os.Getenv(envVar), proxy.URL())

	// The new client should take the proxy into account.
	client, err := backend.NewClient(config.BackendHTTPAPIBaseURL())
	require.Equal(t, err, nil)
	// Perform a request that should go through the proxy.
	request := api.NewPopulatedAppLoginRequest(popr, false)
	_, err = client.AppLogin(request, "my-token")
	// A request has been received:
	//require.NotEqual(t, len(back.ReceivedRequests()), 0, "0 request received")
	require.NotEqual(t, len(proxy.ReceivedRequests()), 0, "0 request received")
}
