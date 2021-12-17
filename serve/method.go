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
	if m.desc.IsStreamingClient() {
		return m.streamingClientCall(ss)
	}
	return m.unaryClientCall(ss)
}

func (m method) unaryClientCall(ss grpc.ServerStream) error {
	// Handle unary client (request), with either unary or streaming server (response).
	req := dynamicpb.NewMessage(m.desc.Input())
	if err := ss.RecvMsg(req); err != nil {
		return err
	}

	input, err := makeInputJSON(req)
	if err != nil {
		return err
	}

	return m.evalJsonnet(input, ss)
}

func (m method) streamingClientCall(ss grpc.ServerStream) error {
	var stream []*dynamicpb.Message
	for {
		msg := dynamicpb.NewMessage(m.desc.Input())
		if err := ss.RecvMsg(msg); err != nil {
			if !errors.Is(err, io.EOF) {
				return err
			}
			break
		}
		if !m.desc.IsStreamingServer() {
			// For client-streaming, we just collect all the messages on
			// the input stream to pass once to jsonnet.
			stream = append(stream, msg)
			continue
		}

		// For bidirectional streaming, we call jsonnet once for each message
		// on the input stream and stream out the results.
		input, err := makeInputJSON(msg)
		if err != nil {
			return err
		}
		if err := m.evalJsonnet(input, ss); err != nil {
			return err
		}
	}

	// For bidirectional streaming, call jsonnet one last time with a null
	// request so it knows end-of-stream has been reached.
	input := "{request: null}"

	if !m.desc.IsStreamingServer() {
		var err error
		input, err = makeStreamingInputJSON(stream)
		if err != nil {
			return err
		}
	}

	return m.evalJsonnet(input, ss)
}

func (m method) evalJsonnet(input string, ss grpc.ServerStream) error {
	vm := jsonnet.MakeVM()
	// TODO(camh): Add jsonnext.Importer
	vm.TLACode("input", input)
	output, err := vm.EvaluateFile(m.filename)
	if err != nil {
		return err
	}

	stream, err := parseOutputJSON(output, m.desc)
	if err != nil {
		return err
	}
	for _, resp := range stream {
		if err := ss.SendMsg(resp); err != nil {
			return err
		}
	}
	return nil
}

type request struct {
	Request json.RawMessage   `json:"request,omitempty"`
	Stream  []json.RawMessage `json:"stream,omitempty"`
}

type response struct {
	Response json.RawMessage   `json:"response"`
	Stream   []json.RawMessage `json:"stream"`
	Status   json.RawMessage   `json:"status"`
}

func makeInputJSON(msg *dynamicpb.Message) (string, error) {
	mo := protojson.MarshalOptions{EmitUnpopulated: true}
	b, err := mo.Marshal(msg)
	if err != nil {
		return "", err
	}
	v := request{Request: b}
	input, err := json.Marshal(&v)
	if err != nil {
		return "", err
	}

	return string(input), nil
}

func makeStreamingInputJSON(stream []*dynamicpb.Message) (string, error) {
	mo := protojson.MarshalOptions{EmitUnpopulated: true}
	v := request{Stream: make([]json.RawMessage, 0, len(stream))}
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

func parseOutputJSON(output string, desc protoreflect.MethodDescriptor) ([]*dynamicpb.Message, error) {
	v := response{}
	if err := json.Unmarshal([]byte(output), &v); err != nil {
		return nil, err
	}

	if len(v.Status) > 0 {
		if len(v.Stream) > 0 || v.Response != nil {
			return nil, errors.New("method cannot return a response/stream and status")
		}
		var s statuspb.Status
		if err := protojson.Unmarshal(v.Status, &s); err != nil {
			return nil, err
		}
		return nil, status.ErrorProto(&s)
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

	stream := make([]*dynamicpb.Message, 0, len(v.Stream))
	for _, jsonMsg := range v.Stream {
		msg := dynamicpb.NewMessage(desc.Output())
		if err := protojson.Unmarshal(jsonMsg, msg); err != nil {
			return nil, err
		}
		stream = append(stream, msg)
	}
	return stream, nil
}
