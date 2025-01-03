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
	"google.golang.org/genproto/googleapis/api/annotations"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestHTTP(t *testing.T) {
	withLogger := serve.WithLogger(log.DiscardLogger)
	ts := serve.NewTestServer(serve.JsonnetEvaluator(), os.DirFS("testdata/greet"), withLogger)
	defer ts.Stop()

	h := NewServer(ts.Files, ts.UnknownHandler, log.DiscardLogger, nil, http.NotFoundHandler())
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
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
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

	t.Run("return 404 for invalid path", func(t *testing.T) {
		req, err := http.NewRequest("GET", url, nil)
		require.NoError(t, err)
		req.Header.Set("Accept", "application/json; charset=utf-8")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestHTTPRuleInterpolation(t *testing.T) {
	logger := log.NewLogger(io.Discard, log.LogLevelError)
	withLogger := serve.WithLogger(logger)
	ts := serve.NewTestServer(serve.JsonnetEvaluator(), os.DirFS("testdata/httpgreet"), withLogger)
	defer ts.Stop()

	tmpl := []*annotations.HttpRule{
		{Pattern: &annotations.HttpRule_Post{Post: "/post/{package}.{service}/{method}"}, Body: "*"},
		{Pattern: &annotations.HttpRule_Get{Get: "/get/{method}"}},
	}
	h := NewServer(ts.Files, ts.UnknownHandler, logger, tmpl, http.NotFoundHandler())
	ts.SetHTTPHandler(h)

	u := "http://" + ts.Addr() + "/get/SimpleHello"
	req, err := http.NewRequest("GET", u, nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "application/json; charset=utf-8")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	raw, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()
	expected := `{"greeting": "Simply, hello, "}`
	require.JSONEq(t, expected, string(raw))

	u = "http://" + ts.Addr() + "/post/httpgreet.HttpGreeter/SimpleHello"
	req, err = http.NewRequest("POST", u, strings.NewReader(`{"first_name": "fox"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	raw, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()
	expected = `{"greeting": "Simply, hello, fox"}`
	require.JSONEq(t, expected, string(raw))
}
