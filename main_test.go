package main

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"foxygo.at/jig/log"
	"foxygo.at/jig/serve"
	"foxygo.at/jig/serve/httprule"
	"github.com/stretchr/testify/require"
)

func TestHTTPRuleServer(t *testing.T) {
	c := cmdServe{
		Proto:     []string{"httpgreet/httpgreet.proto"},
		ProtoPath: []string{"proto"},
	}
	logger := log.NewLogger(io.Discard, log.LogLevelError)
	opts, err := c.getServerOptions(logger)
	require.NoError(t, err)

	ts := serve.NewUnstartedTestServer(serve.JsonnetEvaluator(), os.DirFS("serve/testdata/httpgreet"), opts...)
	ts.SetHTTPHandler(httprule.NewServer(ts.Files, ts.UnknownHandler, logger, nil))
	ts.Start()
	defer ts.Stop()

	baseURL := "http://" + ts.Addr()
	resp, err := http.Get(baseURL + "/api/greet/hello/Dolly")
	require.NoError(t, err)
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"greeting":"httpgreet: Hello, Dolly"}`, string(b))

	resp, err = http.Post(baseURL+"/api/greet/hello", "application/json", strings.NewReader(`{"firstName": "Kitty"}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	b, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"greeting":"Thanks for the post, Kitty"}`, string(b))

	resp, err = http.Post(baseURL+"/api/greet/world", "application/json", strings.NewReader(`{}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	b, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, `{"greeting":"Thanks for the post and the path, world"}`, string(b))
}
