package serve

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"

	"github.com/google/go-jsonnet"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type method struct {
	desc     protoreflect.MethodDescriptor
	filename string
}

func newMethod(md protoreflect.MethodDescriptor, methodDir string) method {
	pkg, svc := md.ParentFile().Package(), md.Parent().Name()
	filename := fmt.Sprintf("%s.%s.%s.jsonnet", pkg, svc, md.Name())
	return method{
		desc:     md,
		filename: path.Join(methodDir, filename),
	}
}

func (m method) fullMethod() string {
	return fmt.Sprintf("/%s.%s/%s", m.desc.ParentFile().Package(), m.desc.Parent().Name(), m.desc.Name())
}

func (m method) call(ss grpc.ServerStream) error {
	switch {
	case m.desc.IsStreamingClient() && m.desc.IsStreamingServer():
		return m.streamingBidiCall(ss)
	case m.desc.IsStreamingClient():
		return m.streamingClientCall(ss)
	default: // handle both unary and streaming-server
		return m.unaryClientCall(ss)
	}
}

func (m method) unaryClientCall(ss grpc.ServerStream) error {
	// Handle unary client (request), with either unary or streaming server (response).
	md, _ := metadata.FromIncomingContext(ss.Context())
	req := dynamicpb.NewMessage(m.desc.Input())
	if err := ss.RecvMsg(req); err != nil {
		return err
	}

	input, err := makeInputJSON(req, md)
	if err != nil {
		return err
	}

	return m.evalJsonnet(input, ss)
}

func (m method) streamingClientCall(ss grpc.ServerStream) error {
	md, _ := metadata.FromIncomingContext(ss.Context())
	var stream []*dynamicpb.Message
	for {
		msg := dynamicpb.NewMessage(m.desc.Input())
		if err := ss.RecvMsg(msg); err != nil {
			if !errors.Is(err, io.EOF) {
				return err
			}
			break
		}
		stream = append(stream, msg)
	}

	input, err := makeStreamingInputJSON(stream, md)
	if err != nil {
		return err
	}

	return m.evalJsonnet(input, ss)
}

func (m method) streamingBidiCall(ss grpc.ServerStream) error {
	md, _ := metadata.FromIncomingContext(ss.Context())
	for {
		msg := dynamicpb.NewMessage(m.desc.Input())
		if err := ss.RecvMsg(msg); err != nil {
			if !errors.Is(err, io.EOF) {
				return err
			}
			break
		}

		// For bidirectional streaming, we call jsonnet once for each message
		// on the input stream and stream out the results.
		input, err := makeInputJSON(msg, md)
		if err != nil {
			return err
		}
		if err := m.evalJsonnet(input, ss); err != nil {
			return err
		}
	}
	return nil
}

func (m method) evalJsonnet(input string, ss grpc.ServerStream) error {
	vm := jsonnet.MakeVM()
	// TODO(camh): Add jsonnext.Importer
	vm.TLACode("input", input)
	output, err := vm.EvaluateFile(m.filename)
	if err != nil {
		return err
	}

	result, err := parseOutputJSON(output, m.desc)
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

func makeInputJSON(msg *dynamicpb.Message, md metadata.MD) (string, error) {
	v := request{Header: md, Request: []byte("null")}
	if msg != nil {
		mo := protojson.MarshalOptions{EmitUnpopulated: true}
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

func makeStreamingInputJSON(stream []*dynamicpb.Message, md metadata.MD) (string, error) {
	mo := protojson.MarshalOptions{EmitUnpopulated: true}
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

func parseOutputJSON(output string, desc protoreflect.MethodDescriptor) (*methodResult, error) {
	v := response{}
	if err := json.Unmarshal([]byte(output), &v); err != nil {
		return nil, err
	}

	result := &methodResult{
		header:  v.Header,
		trailer: v.Trailer,
	}

	if len(v.Status) > 0 {
		if len(v.Stream) > 0 || v.Response != nil {
			return nil, errors.New("method cannot return a response/stream and status")
		}
		var s statuspb.Status
		if err := protojson.Unmarshal(v.Status, &s); err != nil {
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
		if err := protojson.Unmarshal(jsonMsg, msg); err != nil {
			return nil, err
		}
		result.stream = append(result.stream, msg)
	}
	return result, nil
}
