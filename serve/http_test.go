package serve

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"foxygo.at/jig/pb/greet"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestHTTP(t *testing.T) {
	ts := newTestServer()
	defer ts.Stop()

	body := `{"first_name": "Stranger"}`
	url := fmt.Sprintf("http://%s/api/greet/hello", ts.Addr())

	t.Run("accept JSON response", func(t *testing.T) {
		resp, err := http.Post(url, "application/json; charset=utf-8", strings.NewReader(body))
		require.NoError(t, err)

		respPb := &greet.HelloResponse{}
		raw, err := ioutil.ReadAll(resp.Body)
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
		defer resp.Body.Close()
		require.NoError(t, proto.Unmarshal(raw, respPb))

		expected := &greet.HelloResponse{Greeting: "💃 jig [unary]: Hello Stranger"}
		require.Truef(t, proto.Equal(expected, respPb), "expected: %s, \nactual: %s", expected, respPb)
	})
}
