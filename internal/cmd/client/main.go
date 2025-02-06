package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"foxygo.at/jig/pb/greet"
	"github.com/alecthomas/kong"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var version = "v0.0.0"

type config struct {
	Address string           `help:"hostname:port" default:"localhost:8080"`
	Stream  string           `short:"s" enum:"none,client,server,bidi" default:"none" help:"Stream requests/responses"`
	Names   []string         `arg:"" help:"Name to greet" default:"🌏"`
	Version kong.VersionFlag `short:"V" help:"Print version information" group:"Other:"`

	out    io.Writer
	client greet.GreeterClient
}

func main() {
	cfg := &config{out: os.Stdout}
	kctx := kong.Parse(cfg, kong.Vars{"version": version})
	err := kctx.Run()
	kctx.FatalIfErrorf(err)
}

func (cfg *config) AfterApply() error {
	if (cfg.Stream == "none" || cfg.Stream == "server") && len(cfg.Names) != 1 {
		return fmt.Errorf("exactly 1 name required with stream %s", cfg.Stream)
	}
	return nil
}

func (cfg *config) Run() error {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	cc, err := grpc.NewClient(cfg.Address, opts...)
	if err != nil {
		return err
	}
	cfg.client = greet.NewGreeterClient(cc)
	defer cc.Close()
	if err := cfg.call(); err != nil {
		return withDetails(err)
	}
	return nil
}

func (cfg *config) call() error {
	switch cfg.Stream {
	case "none", "unary":
		return cfg.callUnary()
	case "client":
		return cfg.callClientStream()
	case "server":
		return cfg.callServerStream()
	case "bidi":
		return cfg.callBidi()
	default:
		return fmt.Errorf("unknown stream type %q", cfg.Stream)
	}
}
func (cfg *config) callUnary() error {
	var header, trailer metadata.MD
	req := &greet.HelloRequest{FirstName: cfg.Names[0]}
	resp, err := cfg.client.Hello(context.Background(), req, grpc.Header(&header), grpc.Trailer(&trailer))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(cfg.out, "Header: %v\nGreeting: %s\nTrailer: %v\n", header, resp.Greeting, trailer)
	return err
}

func (cfg *config) callClientStream() error {
	stream, err := cfg.client.HelloClientStream(context.Background())
	if err != nil {
		return err
	}
	for _, name := range cfg.Names {
		if err := stream.Send(&greet.HelloRequest{FirstName: name}); err != nil {
			return err
		}
	}
	resp, err := stream.CloseAndRecv()
	if err != nil {
		return err
	}
	header, err := stream.Header()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(cfg.out, "Header: %v\nGreeting: %s\nTrailer: %v\n", header, resp.Greeting, stream.Trailer())
	return err
}

func (cfg *config) callServerStream() error {
	req := &greet.HelloRequest{FirstName: cfg.Names[0]}
	stream, err := cfg.client.HelloServerStream(context.Background(), req)
	if err != nil {
		return err
	}
	header, err := stream.Header()
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cfg.out, "Header: %v\n", header); err != nil {
		return err
	}
	defer fmt.Fprintf(cfg.out, "Trailer: %v\n", stream.Trailer())

	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(cfg.out, "Greeting: %s\n", resp.Greeting)
		if err != nil {
			return err
		}
	}
	return nil
}

func (cfg *config) callBidi() error {
	errgrp, ctx := errgroup.WithContext(context.Background())

	stream, err := cfg.client.HelloBidiStream(ctx)
	if err != nil {
		return err
	}

	// concurrently run each direction of the stream.
	errgrp.Go(func() error {
		for _, name := range cfg.Names {
			req := &greet.HelloRequest{FirstName: name}
			if err := stream.Send(req); err != nil {
				return err
			}
		}
		return stream.CloseSend()
	})
	errgrp.Go(func() error {
		header, err := stream.Header()
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(cfg.out, "Header: %v\n", header)
		if err != nil {
			return err
		}
		defer fmt.Fprintf(cfg.out, "Trailer: %v\n", stream.Trailer())
		for {
			resp, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return err
			}
			fmt.Fprintf(cfg.out, "Greeting: %s\n", resp.Greeting)
		}
	})
	return errgrp.Wait()
}

func withDetails(err error) error {
	if st, ok := status.FromError(err); ok {
		return detailStatusErr{status: st}
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
