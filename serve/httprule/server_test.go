package httprule

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"foxygo.at/jig/log"
	"foxygo.at/jig/pb/greet"
	"foxygo.at/jig/serve"
	"github.com/stretchr/testify/require"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestHTTP(t *testing.T) {
	withLogger := serve.WithLogger(log.NewLogger(io.Discard, log.LogLevelError))
	ts := serve.NewTestServer(serve.JsonnetEvaluator(), os.DirFS("testdata/greet"), withLogger)
	defer ts.Stop()

	h := NewServer(ts.Files, ts.UnknownHandler)
	ts.SetHTTPHandler(h)

	body := `{"first_name": "Stranger"}`
	url := fmt.Sprintf("http://%s/api/greet/hello", ts.Addr())

	t.Run("accept JSON response", func(t *testing.T) {
		resp, err := http.Post(url, "application/json; charset=utf-8", strings.NewReader(body))
		require.NoError(t, err)

		respPb := &greet.HelloResponse{}
		raw, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.NoError(t, protojson.Unmarshal(raw, respPb))

		expected := &greet.HelloResponse{Greeting: "💃 jig [unary]: Hello Stranger"}
		require.Truef(t, proto.Equal(expected, respPb), "expected: %s, \nactual: %s", expected, respPb)
	})

	t.Run("accept binary response", func(t *testing.T) {
		req, err := http.NewRequest("POST", url, strings.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Accept", "application/x-protobuf; charset=utf-8")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		respPb := &greet.HelloResponse{}
		raw, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.NoError(t, proto.Unmarshal(raw, respPb))

		expected := &greet.HelloResponse{Greeting: "💃 jig [unary]: Hello Stranger"}
		require.Truef(t, proto.Equal(expected, respPb), "expected: %s, \nactual: %s", expected, respPb)
	})

	t.Run("converts error responses to HTTP", func(t *testing.T) {
		badRequestBody := `{"first_name": "Bart"}`
		req, err := http.NewRequest("POST", url, strings.NewReader(badRequestBody))
		require.NoError(t, err)
		req.Header.Set("Accept", "application/json; charset=utf-8")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)

		respPb := &statuspb.Status{}
		raw, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.NoError(t, protojson.Unmarshal(raw, respPb))

		respPb.Details = nil
		expected := &statuspb.Status{Code: 3, Message: "💃 jig [unary]: eat my shorts"}
		require.Truef(t, proto.Equal(expected, respPb), "expected: %s, \nactual: %s", expected, respPb)
	})
}
