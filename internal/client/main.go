package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"foxygo.at/pony/pb/echo"
	"github.com/alecthomas/kong"
	"google.golang.org/grpc"
)

var version = "v0.0.0"

type config struct {
	Address string           `help:"hostname:port" default:"localhost:8080"`
	Message string           `arg:"" help:"message to send" default:"Hello 🌏"`
	Version kong.VersionFlag `short:"V" help:"Print version information" group:"Other:"`

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
	resp, err := client.Hello(context.Background(), &echo.HelloRequest{Message: cfg.Message})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(cfg.out, "Response: %s\n", resp.Response)
	return err
}
