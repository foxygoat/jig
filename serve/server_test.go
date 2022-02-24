package serve

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"foxygo.at/jig/internal/client"
	"foxygo.at/jig/log"
	"foxygo.at/jig/pb/greet"
	"github.com/stretchr/testify/require"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func newTestServer() *TestServer {
	withLogger := WithLogger(log.NewLogger(io.Discard, log.LogLevelError))
	return NewTestServer(JsonnetEvaluator(), os.DirFS("testdata/greet"), withLogger)
}

type testCase struct {
	names []string
	want  string
}

func TestGreeterSample(t *testing.T) {
	ts := newTestServer()
	defer ts.Stop()

	c, err := client.New(ts.Addr())
	require.NoError(t, err)
	defer c.Close()

	out := &bytes.Buffer{}

	unaryWant := `
Header: map[content-type:[application/grpc] trailer:[Grpc-Status Grpc-Message Grpc-Status-Details-Bin]]
Greeting: 💃 jig [unary]: Hello 🌏
Trailer: map[]`
	clientWant := `
Header: map[content-type:[application/grpc] count:[3] trailer:[Grpc-Status Grpc-Message Grpc-Status-Details-Bin]]
Greeting: 💃 jig [client]: Hello 1 and 2 and 3
Trailer: map[size:[35]]`
	serverWant := `
Header: map[content-type:[application/grpc] trailer:[Grpc-Status Grpc-Message Grpc-Status-Details-Bin]]
Greeting: 💃 jig [server]: Hello Stranger
Greeting: 💃 jig [server]: Goodbye Stranger
Trailer: map[]`
	bidiWant := `
Header: map[content-type:[application/grpc] trailer:[Grpc-Status Grpc-Message Grpc-Status-Details-Bin]]
Greeting: 💃 jig [bidi]: Hello a b c
Trailer: map[]`

	tests := map[string]testCase{
		"unary":  {names: []string{"🌏"}, want: unaryWant},
		"client": {names: []string{"1", "2", "3"}, want: clientWant},
		"server": {names: []string{"Stranger"}, want: serverWant},
		"bidi":   {names: []string{"a b c"}, want: bidiWant},
	}

	for name, tc := range tests {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			out.Reset()
			err := c.Call(out, tc.names, name)
			require.NoError(t, err)
			want := tc.want[1:] + "\n"
			require.Equal(t, want, out.String())
		})
	}
}

type testCaseStatus struct {
	names   []string
	want    string
	stream  string
	errWant string
}

func TestGreeterSampleStatus(t *testing.T) {
	ts := newTestServer()
	defer ts.Stop()

	c, err := client.New(ts.Addr())
	require.NoError(t, err)
	defer c.Close()

	out := &bytes.Buffer{}

	unaryWant := `
Header: map[content-type:[application/grpc] eat:[my shorts] trailer:[Grpc-Status Grpc-Message Grpc-Status-Details-Bin]]
Trailer: map[a:[cow] dont:[have]]`
	unaryErrWant := `
rpc error: code = InvalidArgument desc = 💃 jig [unary]: eat my shorts
seconds:42
[google.api.http]:{post:"/api/greet/hello"}`
	bidiWant := `
Header: map[content-type:[application/grpc] trailer:[Grpc-Status Grpc-Message Grpc-Status-Details-Bin]]
Greeting: 💃 jig [bidi]: Hello 1
Trailer: map[]`
	bidiErrWant := " rpc error: code = Unknown desc = transport: the stream is done or WriteHeader was already called"
	bidiWant2 := `
Header: map[content-type:[application/grpc] eat:[his shorts] trailer:[Grpc-Status Grpc-Message Grpc-Status-Details-Bin]]
Trailer: map[]`
	bidiErrWant2 := " rpc error: code = InvalidArgument desc = 💃 jig [bidi]: eat my shorts"
	tests := map[string]testCaseStatus{
		"unary": {names: []string{"Bart"}, want: unaryWant, errWant: unaryErrWant, stream: "unary"},
		"bidi":  {names: []string{"1", "Bart", "3"}, want: bidiWant, errWant: bidiErrWant, stream: "bidi"},
		"bidi2": {names: []string{"Bart"}, want: bidiWant2, errWant: bidiErrWant2, stream: "bidi"},
		"bidi3": {names: []string{"1", "Bart"}, want: bidiWant, errWant: bidiErrWant, stream: "bidi"},
	}
	for name, tc := range tests {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			out.Reset()
			err := c.Call(out, tc.names, tc.stream)
			require.Error(t, err)
			require.Equal(t, tc.errWant[1:], err.Error())
			want := tc.want[1:] + "\n"
			require.Equal(t, want, out.String())
		})
	}
}

//go:embed testdata/greet
var embedFS embed.FS

func TestGreeterEmbedFS(t *testing.T) {
	methodFS, err := fs.Sub(embedFS, "testdata/greet")
	require.NoError(t, err)
	ts := NewTestServer(JsonnetEvaluator(), methodFS)
	defer ts.Stop()

	c, err := client.New(ts.Addr())
	require.NoError(t, err)
	defer c.Close()

	out := &bytes.Buffer{}

	want := `Header: map[content-type:[application/grpc] trailer:[Grpc-Status Grpc-Message Grpc-Status-Details-Bin]]
Greeting: 💃 jig [unary]: Hello 🌏
Trailer: map[]
`
	err = c.Call(out, []string{"🌏"}, "unary")
	require.NoError(t, err)
	require.Equal(t, want, out.String())
}

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
