package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"foxygo.at/jig/pb/greet"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Client struct {
	cc            *grpc.ClientConn
	greeterClient greet.GreeterClient
}

func New(addr string) (*Client, error) {
	cc, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	c := &Client{
		cc:            cc,
		greeterClient: greet.NewGreeterClient(cc),
	}
	return c, nil
}

func (c *Client) Close() error {
	if c.cc != nil {
		return c.cc.Close()
	}
	return nil
}

func (c *Client) Call(w io.Writer, names []string, streamType string) error {
	if (streamType == "none" || streamType == "unary" || streamType == "server") && len(names) != 1 {
		return fmt.Errorf("exactly 1 name required with stream %s", streamType)
	}
	var err error
	switch streamType {
	case "none", "unary":
		err = c.CallUnary(w, names[0])
	case "client":
		err = c.CallClientStream(w, names)
	case "server":
		err = c.CallServerStream(w, names[0])
	case "bidi":
		err = c.CallBidiStream(w, names)
	}
	return StatusWithDetails(err)
}

func (c *Client) CallUnary(w io.Writer, name string) error {
	var header, trailer metadata.MD
	req := &greet.HelloRequest{FirstName: name}
	resp, err := c.greeterClient.Hello(context.Background(), req, grpc.Header(&header), grpc.Trailer(&trailer))
	fmt.Fprintf(w, "Header: %v\n", header)
	if err == nil {
		_, err = fmt.Fprintf(w, "Greeting: %s\n", resp.Greeting)
	}
	fmt.Fprintf(w, "Trailer: %v\n", trailer)
	return err
}

func (c *Client) CallClientStream(w io.Writer, names []string) error {
	stream, err := c.greeterClient.HelloClientStream(context.Background())
	if err != nil {
		return err
	}
	for _, name := range names {
		if err := stream.Send(&greet.HelloRequest{FirstName: name}); err != nil {
			return err
		}
	}
	resp, rerr := stream.CloseAndRecv()
	header, err := stream.Header()
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "Header: %v\n", header)
	defer fmt.Fprintf(w, "Trailer: %v\n", stream.Trailer())

	if rerr != nil {
		return rerr
	}
	_, err = fmt.Fprintf(w, "Greeting: %s\n", resp.Greeting)
	return err
}

func (c *Client) CallServerStream(w io.Writer, name string) error {
	req := &greet.HelloRequest{FirstName: name}
	stream, err := c.greeterClient.HelloServerStream(context.Background(), req)
	if err != nil {
		return err
	}

	header, err := stream.Header()
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "Header: %v\n", header)
	defer fmt.Fprintf(w, "Trailer: %v\n", stream.Trailer())

	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "Greeting: %s\n", resp.Greeting)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) CallBidiStream(w io.Writer, names []string) error {
	errgrp, ctx := errgroup.WithContext(context.Background())

	stream, err := c.greeterClient.HelloBidiStream(ctx)
	if err != nil {
		return err
	}

	// concurrently run each direction of the stream.
	errgrp.Go(func() error {
		for _, name := range names {
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
		fmt.Fprintf(w, "Header: %v\n", header)
		defer fmt.Fprintf(w, "Trailer: %v\n", stream.Trailer())
		for {
			resp, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return err
			}
			fmt.Fprintf(w, "Greeting: %s\n", resp.Greeting)
		}
	})
	return errgrp.Wait()
}

func StatusWithDetails(err error) error {
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
