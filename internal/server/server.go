package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"foxygo.at/pony/pb/echo"
)

type server struct {
	echo.UnimplementedEchoServiceServer
}

func (*server) Hello(_ context.Context, req *echo.HelloRequest) (*echo.HelloResponse, error) {
	resp := &echo.HelloResponse{Response: fmt.Sprintf("%s ... %s", req.Message, req.Message)}
	return resp, nil
}

func (*server) HelloServerStream(req *echo.HelloRequest, stream echo.EchoService_HelloServerStreamServer) error {
	for i := 0; i < 10; i++ {
		err := stream.Send(&echo.HelloResponse{Response: req.Message})
		if err != nil {
			return err
		}
	}
	return nil
}

func (*server) HelloClientStream(stream echo.EchoService_HelloClientStreamServer) error {
	var messages []string
	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		messages = append(messages, req.Message)
	}
	resp := &echo.HelloResponse{Response: "Hello " + strings.Join(messages, " and ")}
	return stream.SendAndClose(resp)
}

func (*server) HelloBiDiStream(stream echo.EchoService_HelloBiDiStreamServer) error {
	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if err := stream.Send(&echo.HelloResponse{Response: "Hello " + req.Message}); err != nil {
			return err
		}
	}
	return nil
}

func newServer() echo.EchoServiceServer {
	return &server{}
}
