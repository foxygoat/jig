package main

import (
	"context"
	"fmt"

	"foxygo.at/pony/pb/echo"
)

type server struct {
	echo.EchoServiceServer
}

func (*server) Hello(_ context.Context, req *echo.HelloRequest) (*echo.HelloResponse, error) {
	resp := &echo.HelloResponse{Response: fmt.Sprintf("%s ... %s", req.Message, req.Message)}
	return resp, nil
}

func (*server) HelloStream(req *echo.HelloStreamRequest, stream echo.EchoService_HelloStreamServer) error {
	for i := 0; i < 10; i++ {
		err := stream.Send(&echo.HelloStreamResponse{Response: req.Message})
		if err != nil {
			return err
		}
	}
	return nil
}

func newServer() echo.EchoServiceServer {
	return &server{}
}
