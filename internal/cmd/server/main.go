package main

import (
	"fmt"
	"net"

	"foxygo.at/jig/pb/greet"
	"github.com/alecthomas/kong"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var version = "v0.0.0"

type config struct {
	Address string           `help:"hostname:port" default:"localhost:8080"`
	Version kong.VersionFlag `short:"V" help:"Print version information" group:"Other:"`
}

func main() {
	cfg := &config{}
	kctx := kong.Parse(cfg, kong.Vars{"version": version})
	fmt.Println("starting server server on", cfg.Address)
	err := run(cfg.Address)
	kctx.FatalIfErrorf(err)
}

func run(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer()
	reflection.Register(grpcServer)
	greet.RegisterGreeterServer(grpcServer, newServer())
	return grpcServer.Serve(lis)
}
