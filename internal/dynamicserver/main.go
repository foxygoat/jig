package main

import (
	"context"
	"fmt"
	"net"

	"github.com/alecthomas/kong"
	"google.golang.org/grpc"
	preflect "google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

var version = "v0.0.0"

type config struct {
	Address string           `help:"hostname:port" default:"localhost:9090"`
	Version kong.VersionFlag `short:"V" help:"Print version information" group:"Other:"`

	Service string `help:"Service name" default:"echo.EchoService"`
	Method  string `help:"Method name" default:"Hello"`
	In      string `help:"String field of request" default:"message"`
	Out     string `help:"String field of response proto" default:"response"`

	inMessageDescriptor  preflect.MessageDescriptor
	outMessageDescriptor preflect.MessageDescriptor
}

func main() {
	cfg := newConfig()
	kctx := kong.Parse(cfg, kong.Vars{"version": version})
	fmt.Println("starting server server on", cfg.Address)
	err := run(cfg)
	kctx.FatalIfErrorf(err)
}

func newConfig() *config {
	in := HelloRequest{}
	out := HelloResponse{}
	return &config{
		inMessageDescriptor:  in.ProtoReflect().Descriptor(),
		outMessageDescriptor: out.ProtoReflect().Descriptor(),
	}
}

func run(cfg *config) error {
	lis, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return err
	}
	s := grpc.NewServer()
	s.RegisterService(desc(cfg), nil)

	return s.Serve(lis)
}

func desc(cfg *config) *grpc.ServiceDesc {
	return &grpc.ServiceDesc{
		ServiceName: cfg.Service, // "echo.EchoService",
		HandlerType: nil,
		Methods: []grpc.MethodDesc{
			{
				MethodName: cfg.Method, // "Hello",
				Handler:    cfg.handler,
			},
		},
	}
}

func (cfg *config) handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	if interceptor != nil {
		return nil, fmt.Errorf("handler with interceptor not implemented")
	}
	in := dynamicpb.NewMessage(cfg.inMessageDescriptor)
	if err := dec(in); err != nil {
		return nil, err
	}
	return cfg.transform(in), nil
}

func (cfg *config) transform(in *dynamicpb.Message) *dynamicpb.Message {
	fd := cfg.inMessageDescriptor.Fields().ByJSONName(cfg.In)
	val := in.Get(fd)

	out := dynamicpb.NewMessage(cfg.outMessageDescriptor)
	fd = cfg.outMessageDescriptor.Fields().ByJSONName(cfg.Out)
	out.Set(fd, val)

	return out
}
