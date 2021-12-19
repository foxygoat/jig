package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"foxygo.at/jig/pb/greet"
)

type server struct {
	greet.UnimplementedGreeterServer
}

func (*server) Hello(_ context.Context, req *greet.HelloRequest) (*greet.HelloResponse, error) {
	resp := &greet.HelloResponse{Greeting: "👷 grpc-gen-go [unary]: Hello " + req.FirstName}
	return resp, nil
}

func (*server) HelloServerStream(req *greet.HelloRequest, stream greet.Greeter_HelloServerStreamServer) error {
	for i := 0; i < 10; i++ {
		err := stream.Send(&greet.HelloResponse{Greeting: fmt.Sprintf("👷 grpc-gen-go [server %d]: Hello %s", i, req.FirstName)})
		if err != nil {
			return err
		}
	}
	return nil
}

func (*server) HelloClientStream(stream greet.Greeter_HelloClientStreamServer) error {
	var names []string
	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		names = append(names, req.FirstName)
	}
	resp := &greet.HelloResponse{Greeting: "👷 grpc-gen-go [client]: Hello " + strings.Join(names, " and ")}
	return stream.SendAndClose(resp)
}

func (*server) HelloBiDiStream(stream greet.Greeter_HelloBiDiStreamServer) error {
	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if err := stream.Send(&greet.HelloResponse{Greeting: "👷 grpc-gen-go [bidi]: Hello " + req.FirstName}); err != nil {
			return err
		}
	}
	return nil
}

func newServer() greet.GreeterServer {
	return &server{}
}
