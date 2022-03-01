package serve

import (
	"encoding/json"
	"errors"
	"io"

	"foxygo.at/jig/registry"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func (s *Server) callMethod(md protoreflect.MethodDescriptor, ss grpc.ServerStream) error {
	switch {
	case md.IsStreamingClient() && md.IsStreamingServer():
		return s.streamingBidiCall(md, ss)
	case md.IsStreamingClient():
		return s.streamingClientCall(md, ss)
	default: // handle both unary and streaming-server
		return s.unaryClientCall(md, ss)
	}
}

func (s *Server) unaryClientCall(md protoreflect.MethodDescriptor, ss grpc.ServerStream) error {
	// Handle unary client (request), with either unary or streaming server (response).
	mdata, _ := metadata.FromIncomingContext(ss.Context())
	req := dynamicpb.NewMessage(md.Input())
	if err := ss.RecvMsg(req); err != nil {
		return err
	}

	input, err := makeInputJSON(req, mdata, s.Files)
	if err != nil {
		return err
	}

	return s.evaluate(md, input, ss, s.Files)
}

func (s *Server) streamingClientCall(md protoreflect.MethodDescriptor, ss grpc.ServerStream) error {
	mdata, _ := metadata.FromIncomingContext(ss.Context())
	var stream []*dynamicpb.Message
	for {
		msg := dynamicpb.NewMessage(md.Input())
		if err := ss.RecvMsg(msg); err != nil {
			if !errors.Is(err, io.EOF) {
				return err
			}
			break
		}
		stream = append(stream, msg)
	}

	input, err := makeStreamingInputJSON(stream, mdata, s.Files)
	if err != nil {
		return err
	}

	return s.evaluate(md, input, ss, s.Files)
}

func (s *Server) streamingBidiCall(md protoreflect.MethodDescriptor, ss grpc.ServerStream) error {
	mdata, _ := metadata.FromIncomingContext(ss.Context())
	for {
		msg := dynamicpb.NewMessage(md.Input())
		if err := ss.RecvMsg(msg); err != nil {
			if !errors.Is(err, io.EOF) {
				return err
			}
			break
		}

		// For bidirectional streaming, we call evaluator once for each message
		// on the input stream and stream out the results.
		input, err := makeInputJSON(msg, mdata, s.Files)
		if err != nil {
			return err
		}
		if err := s.evaluate(md, input, ss, s.Files); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) evaluate(md protoreflect.MethodDescriptor, input string, ss grpc.ServerStream, reg *registry.Files) error {
	output, err := s.eval.Evaluate(string(md.FullName()), input, s.fs)
	if err != nil {
		return err
	}

	result, err := parseOutputJSON(output, md, s.Files)
	if err != nil {
		return err
	}
	if len(result.header) > 0 {
		if err := ss.SetHeader(result.header); err != nil {
			return err
		}
	}
	if len(result.trailer) > 0 {
		ss.SetTrailer(result.trailer)
	}
	if result.status != nil {
		return status.ErrorProto(result.status)
	}
	for _, resp := range result.stream {
		if err := ss.SendMsg(resp); err != nil {
			return err
		}
	}
	return nil
}

type request struct {
	Header  metadata.MD       `json:"header"`
	Request json.RawMessage   `json:"request,omitempty"`
	Stream  []json.RawMessage `json:"stream,omitempty"`
}

type response struct {
	Header   metadata.MD       `json:"header"`
	Trailer  metadata.MD       `json:"trailer"`
	Response json.RawMessage   `json:"response"`
	Stream   []json.RawMessage `json:"stream"`
	Status   json.RawMessage   `json:"status"`
}

type methodResult struct {
	header  metadata.MD
	trailer metadata.MD
	stream  []*dynamicpb.Message
	status  *statuspb.Status
}

func makeInputJSON(msg *dynamicpb.Message, md metadata.MD, reg *registry.Files) (string, error) {
	v := request{Header: md, Request: []byte("null")}
	if msg != nil {
		mo := protojson.MarshalOptions{EmitUnpopulated: true, Resolver: reg}
		b, err := mo.Marshal(msg)
		if err != nil {
			return "", err
		}
		v.Request = b
	}
	input, err := json.Marshal(&v)
	if err != nil {
		return "", err
	}

	return string(input), nil
}

func makeStreamingInputJSON(stream []*dynamicpb.Message, md metadata.MD, reg *registry.Files) (string, error) {
	mo := protojson.MarshalOptions{EmitUnpopulated: true, Resolver: reg}
	v := request{Header: md, Stream: make([]json.RawMessage, 0, len(stream))}
	for _, msg := range stream {
		b, err := mo.Marshal(msg)
		if err != nil {
			return "", err
		}
		v.Stream = append(v.Stream, b)
	}

	input, err := json.Marshal(&v)
	if err != nil {
		return "", err
	}

	return string(input), nil
}

func parseOutputJSON(output string, desc protoreflect.MethodDescriptor, reg *registry.Files) (*methodResult, error) {
	v := response{}
	if err := json.Unmarshal([]byte(output), &v); err != nil {
		return nil, err
	}

	result := &methodResult{
		header:  v.Header,
		trailer: v.Trailer,
	}

	uo := protojson.UnmarshalOptions{Resolver: reg}

	if len(v.Status) > 0 {
		if len(v.Stream) > 0 || v.Response != nil {
			return nil, errors.New("method cannot return a response/stream and status")
		}
		var s statuspb.Status
		if err := uo.Unmarshal(v.Status, &s); err != nil {
			return nil, err
		}
		result.status = &s
		return result, nil
	}

	// Validate result. A streaming server can return a nil slice.
	switch {
	case !desc.IsStreamingServer() && len(v.Stream) > 0:
		return nil, errors.New("unary server method returned a stream")
	case !desc.IsStreamingServer() && v.Response == nil:
		return nil, errors.New("unary server method did not return a response")
	case desc.IsStreamingServer() && v.Response != nil:
		return nil, errors.New("server streaming method returned singular response")
	}

	if !desc.IsStreamingServer() {
		// Put the singular response into the (empty) stream to return a slice of one element.
		v.Stream = append(v.Stream, v.Response)
	}

	result.stream = make([]*dynamicpb.Message, 0, len(v.Stream))
	for _, jsonMsg := range v.Stream {
		msg := dynamicpb.NewMessage(desc.Output())
		if err := uo.Unmarshal(jsonMsg, msg); err != nil {
			return nil, err
		}
		result.stream = append(result.stream, msg)
	}
	return result, nil
}
