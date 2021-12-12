package serve

import (
	"context"
	"fmt"
	"testing"

	"foxygo.at/pony/pb/echo"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestEchoServer(t *testing.T) {
	s := &TestServer{
		Server{
			Listen:    "localhost:0",
			MethodDir: "testdata",
			ProtoSet:  "testdata/all.pb",
		},
	}
	require.NoError(t, s.Start())
	defer s.Stop()

	conn, err := grpc.Dial(s.Addr(), grpc.WithInsecure())
	require.NoError(t, err)
	defer conn.Close()
	client := echo.NewEchoServiceClient(conn)

	ctx := context.Background()
	resp, err := client.Hello(ctx, &echo.HelloRequest{Message: "🦊"})
	require.NoError(t, err)
	require.Equal(t, "Hello 🦊", resp.Response)

	stream, err := client.HelloClientStream(ctx)
	fmt.Println("stream set up")
	require.NoError(t, err)
	for i := 1; i < 4; i++ {
		fmt.Println(i)
		msg := fmt.Sprintf("%d", i)
		require.NoError(t, stream.Send(&echo.HelloRequest{Message: msg}))
	}
	fmt.Println("sending close")

	resp, err = stream.CloseAndRecv()
	fmt.Println("done")

	require.NoError(t, err)
	require.Equal(t, "Hello 1 and 2 and 3", resp.Response)
}
