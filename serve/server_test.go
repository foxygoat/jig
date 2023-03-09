package serve

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"foxygo.at/jig/internal/client"
	"foxygo.at/jig/log"
	"github.com/stretchr/testify/require"
)

func newTestServer() *TestServer {
	withLogger := WithLogger(log.DiscardLogger)
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
Header: map[content-type:[application/grpc]]
Greeting: 💃 jig [unary]: Hello 🌏
Trailer: map[]`
	clientWant := `
Header: map[content-type:[application/grpc] count:[3]]
Greeting: 💃 jig [client]: Hello 1 and 2 and 3
Trailer: map[size:[35]]`
	serverWant := `
Header: map[content-type:[application/grpc]]
Greeting: 💃 jig [server]: Hello Stranger
Greeting: 💃 jig [server]: Goodbye Stranger
Trailer: map[]`
	bidiWant := `
Header: map[content-type:[application/grpc]]
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
Header: map[content-type:[application/grpc] eat:[my shorts]]
Trailer: map[a:[cow] dont:[have]]`
	unaryErrWant := `
rpc error: code = InvalidArgument desc = 💃 jig [unary]: eat my shorts
seconds:42
[google.api.http]:{post:"/api/greet/hello"}`
	bidiWant := `
Header: map[content-type:[application/grpc]]
Greeting: 💃 jig [bidi]: Hello 1
Trailer: map[]`
	bidiErrWant := " rpc error: code = Internal desc = transport: SendHeader called multiple times"
	bidiWant2 := `
Header: map[content-type:[application/grpc] eat:[his shorts]]
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

	want := `Header: map[content-type:[application/grpc]]
Greeting: 💃 jig [unary]: Hello 🌏
Trailer: map[]
`
	err = c.Call(out, []string{"🌏"}, "unary")
	require.NoError(t, err)
	require.Equal(t, want, out.String())
}

func TestHTTPHandler(t *testing.T) {
	ts := newTestServer()
	defer ts.Stop()

	mux := http.NewServeMux()
	mux.HandleFunc("/foo", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("bar")) //nolint:errcheck
	})
	ts.SetHTTPHandler(mux)

	c, err := client.New(ts.Addr())
	require.NoError(t, err)
	defer c.Close()

	out := &bytes.Buffer{}

	// The http.Handler implementation of grpc.Server adds the "trailer" headers
	// to the response. The built-in implementation does not.
	grpcWant := `
Header: map[content-type:[application/grpc] trailer:[Grpc-Status Grpc-Message Grpc-Status-Details-Bin]]
Greeting: 💃 jig [unary]: Hello 🌏
Trailer: map[]`

	// Test that gRPC calls work when an http handler is installed
	err = c.Call(out, []string{"🌏"}, "unary")
	require.NoError(t, err)
	want := grpcWant[1:] + "\n"
	require.Equal(t, want, out.String())

	// Test that the http handler is called
	url := fmt.Sprintf("http://%s/foo", ts.Addr())
	resp, err := http.Get(url)
	require.NoError(t, err)
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "bar", string(body))
}
