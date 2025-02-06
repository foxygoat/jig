package serve

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"testing"

	"foxygo.at/jig/log"
	"foxygo.at/jig/pb/greet"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/durationpb"
)

func newTestServer() *TestServer {
	withLogger := WithLogger(log.DiscardLogger)
	return NewTestServer(JsonnetEvaluator(), os.DirFS("testdata/greet"), withLogger)
}

func TestGreeterUnary(t *testing.T) {
	ts := newTestServer()
	defer ts.Stop()
	c := newGreeterClient(t, ts.Addr())
	defer c.Close()

	var header, trailer metadata.MD
	req := &greet.HelloRequest{FirstName: "🌏"}
	resp, err := c.Hello(context.Background(), req, grpc.Header(&header), grpc.Trailer(&trailer))
	require.NoError(t, err)
	require.Equal(t, "💃 jig [unary]: Hello 🌏", resp.Greeting)
	require.Equal(t, []string{"application/grpc"}, header.Get("content-type"))
	require.Empty(t, trailer)
}

func TestGreeterUnaryWithStatusErr(t *testing.T) {
	ts := newTestServer()
	defer ts.Stop()
	c := newGreeterClient(t, ts.Addr())
	defer c.Close()

	var header, trailer metadata.MD
	req := &greet.HelloRequest{FirstName: "Bart"}
	resp, err := c.Hello(context.Background(), req, grpc.Header(&header), grpc.Trailer(&trailer))
	require.Nil(t, resp)
	// header
	require.Equal(t, []string{"application/grpc"}, header.Get("content-type"))
	require.Equal(t, []string{"my", "shorts"}, header.Get("eat"))
	// trailer
	require.Equal(t, []string{"cow"}, trailer.Get("a"))
	require.Equal(t, []string{"have"}, trailer.Get("dont"))
	// error
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, st.Code())
	require.Equal(t, "💃 jig [unary]: eat my shorts", st.Message())
	details := st.Details()
	require.Len(t, details, 2)
	wantDur := &durationpb.Duration{Seconds: 42}
	diff := cmp.Diff(wantDur, details[0], protocmp.Transform())
	require.Empty(t, diff)
	gotOpt := details[1].(*descriptorpb.MethodOptions).String()
	wantOpt := `[google.api.http]:{post:"/api/greet/hello"}`
	require.Equal(t, wantOpt, gotOpt)
}

func TestGreeterClientStream(t *testing.T) {
	ts := newTestServer()
	defer ts.Stop()
	c := newGreeterClient(t, ts.Addr())
	defer c.Close()

	stream, err := c.HelloClientStream(context.Background())
	require.NoError(t, err)

	names := []string{"1", "2", "3"}
	for _, name := range names {
		err := stream.Send(&greet.HelloRequest{FirstName: name})
		require.NoError(t, err)
	}
	resp, err := stream.CloseAndRecv()
	require.NoError(t, err)
	wantGreeting := "💃 jig [client]: Hello 1 and 2 and 3"
	require.Equal(t, wantGreeting, resp.Greeting)

	header, err := stream.Header()
	require.NoError(t, err)
	trailer := stream.Trailer()
	require.Equal(t, []string{"application/grpc"}, header.Get("content-type"))
	require.Equal(t, []string{"3"}, header.Get("count"))
	require.Equal(t, []string{"35"}, trailer.Get("size"))
}

func TestGreeterServerStream(t *testing.T) {
	ts := newTestServer()
	defer ts.Stop()
	c := newGreeterClient(t, ts.Addr())
	defer c.Close()

	req := &greet.HelloRequest{FirstName: "Stranger"}
	stream, err := c.HelloServerStream(context.Background(), req)
	require.NoError(t, err)
	header, err := stream.Header()
	require.NoError(t, err)
	trailer := stream.Trailer()
	var greetings []string
	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		greetings = append(greetings, resp.Greeting)
	}
	require.Equal(t, []string{"💃 jig [server]: Hello Stranger", "💃 jig [server]: Goodbye Stranger"}, greetings)
	require.Equal(t, []string{"application/grpc"}, header.Get("content-type"))
	require.Empty(t, trailer)
}

func TestGreeterBidi(t *testing.T) {
	ts := newTestServer()
	defer ts.Stop()
	c := newGreeterClient(t, ts.Addr())
	defer c.Close()

	stream, err := c.HelloBidiStream(context.Background())
	require.NoError(t, err)

	names := []string{"a", "b", "c"}
	for _, name := range names {
		err := stream.Send(&greet.HelloRequest{FirstName: name})
		require.NoError(t, err)
	}
	require.NoError(t, stream.CloseSend())
	header, err := stream.Header()
	require.NoError(t, err)
	trailer := stream.Trailer()
	var greetings []string
	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		greetings = append(greetings, resp.Greeting)
	}
	require.NoError(t, err)
	require.Equal(t, []string{"💃 jig [bidi]: Hello a", "💃 jig [bidi]: Hello b", "💃 jig [bidi]: Hello c"}, greetings)
	require.Equal(t, []string{"application/grpc"}, header.Get("content-type"))
	require.Empty(t, trailer)
}

func TestGreeterBidiStatusErr(t *testing.T) {
	ts := newTestServer()
	defer ts.Stop()
	c := newGreeterClient(t, ts.Addr())
	defer c.Close()

	stream, err := c.HelloBidiStream(context.Background())
	require.NoError(t, err)

	err = stream.Send(&greet.HelloRequest{FirstName: "Bart"})
	require.NoError(t, err)
	require.NoError(t, stream.CloseSend())
	resp, err := stream.Recv()
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, st.Code())
	require.Equal(t, "💃 jig [bidi]: eat my shorts", st.Message())

	require.Nil(t, resp)
	require.Empty(t, stream.Trailer())
	header, err := stream.Header()
	require.NoError(t, err)
	require.Equal(t, []string{"application/grpc"}, header.Get("content-type"))
	require.Equal(t, []string{"his", "shorts"}, header.Get("eat"))
}

func TestGreeterBidiMultiResponseStatusErr(t *testing.T) {
	ts := newTestServer()
	defer ts.Stop()
	c := newGreeterClient(t, ts.Addr())
	defer c.Close()

	stream, err := c.HelloBidiStream(context.Background())
	require.NoError(t, err)

	err = stream.Send(&greet.HelloRequest{FirstName: "Amigo"})
	require.NoError(t, err)
	err = stream.Send(&greet.HelloRequest{FirstName: "Bart"})
	require.NoError(t, err)
	require.NoError(t, stream.CloseSend())
	resp, err := stream.Recv()
	require.NoError(t, err)
	require.Equal(t, "💃 jig [bidi]: Hello Amigo", resp.Greeting)
	resp, err = stream.Recv()
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Internal, st.Code())
	require.Equal(t, "transport: SendHeader called multiple times", st.Message())

	require.Nil(t, resp)
	require.Empty(t, stream.Trailer())
	header, err := stream.Header()
	require.NoError(t, err)
	require.Equal(t, []string{"application/grpc"}, header.Get("content-type"))
}

//go:embed testdata/greet
var embedFS embed.FS

func TestGreeterEmbedFS(t *testing.T) {
	methodFS, err := fs.Sub(embedFS, "testdata/greet")
	require.NoError(t, err)
	ts := NewTestServer(JsonnetEvaluator(), methodFS)
	defer ts.Stop()
	c := newGreeterClient(t, ts.Addr())
	defer c.Close()

	var header, trailer metadata.MD
	req := &greet.HelloRequest{FirstName: "🌏"}
	resp, err := c.Hello(context.Background(), req, grpc.Header(&header), grpc.Trailer(&trailer))
	require.NoError(t, err)
	require.Equal(t, "💃 jig [unary]: Hello 🌏", resp.Greeting)
	require.Equal(t, []string{"application/grpc"}, header.Get("content-type"))
	require.Empty(t, trailer)
}

func TestHTTPHandler(t *testing.T) {
	ts := NewUnstartedTestServer(JsonnetEvaluator(), os.DirFS("testdata/greet"), WithLogger(log.DiscardLogger))
	mux := http.NewServeMux()
	mux.HandleFunc("/foo", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("bar")) //nolint:errcheck
	})
	ts.SetHTTPHandler(mux)
	ts.Start()
	defer ts.Stop()

	c := newGreeterClient(t, ts.Addr())
	defer c.Close()

	// Test that gRPC calls work when an http handler is installed
	var header, trailer metadata.MD
	req := &greet.HelloRequest{FirstName: "🌏"}
	resp, err := c.Hello(context.Background(), req, grpc.Header(&header), grpc.Trailer(&trailer))
	require.NoError(t, err)
	require.Equal(t, "💃 jig [unary]: Hello 🌏", resp.Greeting)
	// The http.Handler implementation of grpc.Server adds the "trailer" headers
	// to the response. The built-in implementation does not.
	require.Equal(t, []string{"application/grpc"}, header.Get("content-type"))
	require.Equal(t, []string{"Grpc-Status", "Grpc-Message", "Grpc-Status-Details-Bin"}, header.Get("trailer"))
	require.Empty(t, trailer)

	// Test that the http handler is called
	url := fmt.Sprintf("http://%s/foo", ts.Addr())
	httpResp, err := http.Get(url)
	require.NoError(t, err)
	body, err := io.ReadAll(httpResp.Body)
	require.NoError(t, err)
	require.Equal(t, "bar", string(body))
}

type greeterClient struct {
	*grpc.ClientConn
	greet.GreeterClient
}

func newGreeterClient(t *testing.T, addr string) *greeterClient {
	t.Helper()
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	cc, err := grpc.NewClient(addr, dialOpts...)
	require.NoError(t, err)
	gc := greet.NewGreeterClient(cc)
	return &greeterClient{
		ClientConn:    cc,
		GreeterClient: gc,
	}
}
