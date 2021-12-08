package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"foxygo.at/pony/pb/echo"
	"github.com/alecthomas/kong"
	"google.golang.org/grpc"
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
		return runUnary(client, cfg)
	case "client":
		return runClientStream(client, cfg)
	case "server":
		return runServerStream(client, cfg)
	case "bidi":
		return runBiDiStream(client, cfg)
	}
	return nil
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
			return nil
		}
		fmt.Fprintf(cfg.out, "Response: %s\n", resp.Response)
	}
	return nil
}
