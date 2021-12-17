package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"foxygo.at/jig/pb/echo"
	"github.com/alecthomas/kong"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var version = "v0.0.0"

type config struct {
	Address  string           `help:"hostname:port" default:"localhost:8080"`
	Stream   string           `short:"s" enum:"none,client,server,bidi" default:"none" help:"Stream requests/responses"`
	Messages []string         `arg:"" help:"message to send" default:"Hello 🌏"`
	Version  kong.VersionFlag `short:"V" help:"Print version information" group:"Other:"`

	out io.Writer
}

func main() {
	cfg := &config{out: os.Stdout}
	kctx := kong.Parse(cfg, kong.Vars{"version": version})
	err := run(cfg)
	kctx.FatalIfErrorf(err)
}

func run(cfg *config) error {
	conn, err := grpc.Dial(cfg.Address, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()
	client := echo.NewEchoServiceClient(conn)
	switch cfg.Stream {
	case "none":
		err = runUnary(client, cfg)
	case "client":
		err = runClientStream(client, cfg)
	case "server":
		err = runServerStream(client, cfg)
	case "bidi":
		err = runBiDiStream(client, cfg)
	}
	return statusWithDetails(err)
}

func runUnary(client echo.EchoServiceClient, cfg *config) error {
	if len(cfg.Messages) > 1 {
		return errors.New("Only one message allowed for unary client requests")
	}
	resp, err := client.Hello(context.Background(), &echo.HelloRequest{Message: cfg.Messages[0]})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(cfg.out, "Response: %s\n", resp.Response)
	return err
}

func runClientStream(client echo.EchoServiceClient, cfg *config) error {
	stream, err := client.HelloClientStream(context.Background())
	if err != nil {
		return err
	}
	for _, msg := range cfg.Messages {
		if err := stream.Send(&echo.HelloRequest{Message: msg}); err != nil {
			return err
		}
	}
	resp, err := stream.CloseAndRecv()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(cfg.out, "Response: %s\n", resp.Response)
	return err
}

func runServerStream(client echo.EchoServiceClient, cfg *config) error {
	if len(cfg.Messages) > 1 {
		return errors.New("Only one message allowed for unary client requests")
	}
	req := &echo.HelloRequest{Message: cfg.Messages[0]}
	stream, err := client.HelloServerStream(context.Background(), req)
	if err != nil {
		return err
	}
	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(cfg.out, "Response: %s\n", resp.Response)
		if err != nil {
			return err
		}
	}
	return nil
}

func runBiDiStream(client echo.EchoServiceClient, cfg *config) error {
	stream, err := client.HelloBiDiStream(context.Background())
	if err != nil {
		return err
	}
	for _, msg := range cfg.Messages {
		if err := stream.Send(&echo.HelloRequest{Message: msg}); err != nil {
			return err
		}
		// We don't need to run stream.Recv() in a separate goroutine like
		// some bidi methods need as the echo service is synchronous. We
		// send one request, we get one response. For asynchronous bidi
		// streaming methods, this Recv() would likely need to be done
		// concurrently/asynchronously with the Send().
		resp, err := stream.Recv()
		if err != nil {
			// EOF is an error here, because we expect a response
			return err
		}
		fmt.Fprintf(cfg.out, "Response: %s\n", resp.Response)
	}
	return nil
}

func statusWithDetails(err error) error {
	if st, ok := status.FromError(err); ok && st != nil {
		return detailStatusErr{st}
	}
	return err
}

type detailStatusErr struct {
	status *status.Status
}

func (dst detailStatusErr) Error() string {
	details := dst.status.Details()
	if len(details) == 0 {
		return dst.status.Err().Error()
	}
	lines := make([]string, 0, len(details)+1)
	lines = append(lines, dst.status.Err().Error())
	for _, d := range details {
		lines = append(lines, fmt.Sprintf("%v", d))
	}
	return strings.Join(lines, "\n")
}
