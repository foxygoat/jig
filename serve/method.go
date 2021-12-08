package serve

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"

	"github.com/google/go-jsonnet"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type method struct {
	desc     protoreflect.MethodDescriptor
	filename string
}

type serverStream interface {
	SendMsg(m interface{}) error
	RecvMsg(m interface{}) error
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

func (m method) call(ss serverStream) error {
	if m.desc.IsStreamingClient() {
		return m.streamingClientCall(ss)
	}
	return m.unaryClientCall(ss)
}

func (m method) unaryClientCall(ss serverStream) error {
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

func (m method) streamingClientCall(ss serverStream) error {
	var err error
	var input string
	var stream []*dynamicpb.Message
	for {
		msg := dynamicpb.NewMessage(m.desc.Input())
		if err = ss.RecvMsg(msg); err != nil {
			break
		}
		if !m.desc.IsStreamingServer() {
			// For a unary response, we just collect all the messages on
			// the input stream to pass once to jsonnet.
			stream = append(stream, msg)
			continue
		}

		// For a streaming response, we call jsonnet once for each message
		// on the input stream and stream out the results.
		input, err = makeInputJSON(msg)
		if err != nil {
			return err
		}
		if err = m.evalJsonnet(input, ss); err != nil {
			return err
		}
	}

	if !errors.Is(err, io.EOF) {
		return err
	}

	if !m.desc.IsStreamingServer() {
		input, err = makeStreamingInputJSON(stream)
		if err != nil {
			return err
		}
	} else if input == "" {
		// We are a streaming server but have nothing on the input stream.
		// Call jsonnet once with a null request so it can choose to stream
		// messages back if it wants.
		input = "{response: null}"
	}
	return m.evalJsonnet(input, ss)
}

func (m method) evalJsonnet(input string, ss serverStream) error {
	vm := jsonnet.MakeVM()
	// TODO(camh): Add jsonnext.Importer
	fmt.Printf("eval input  = %s\n", input)
	vm.TLACode("input", input)
	output, err := vm.EvaluateFile(m.filename)
	if err != nil {
		return err
	}
	fmt.Printf("eval output = %s\n", output)

	stream, err := parseOutputJSON(output, m.desc)
	for _, resp := range stream { // stream is nil (empty) on error
		if err := ss.SendMsg(resp); err != nil {
			return err
		}
	}
	return err
}

type request struct {
	Request json.RawMessage   `json:"request,omitempty"`
	Stream  []json.RawMessage `json:"stream,omitempty"`
}

type response struct {
	Response json.RawMessage   `json:"response"`
	Stream   []json.RawMessage `json:"stream"`
}

func makeInputJSON(msg *dynamicpb.Message) (string, error) {
	b, err := protojson.Marshal(msg)
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
	v := request{Stream: make([]json.RawMessage, 0, len(stream))}
	for _, msg := range stream {
		b, err := protojson.Marshal(msg)
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

	var stream []*dynamicpb.Message
	switch {
	case desc.IsStreamingServer():
		stream = make([]*dynamicpb.Message, 0, len(v.Stream))
		for _, jsonMsg := range v.Stream {
			msg := dynamicpb.NewMessage(desc.Output())
			if err := protojson.Unmarshal(jsonMsg, msg); err != nil {
				return nil, err
			}
			stream = append(stream, msg)
		}

	case v.Response != nil:
		msg := dynamicpb.NewMessage(desc.Output())
		if err := protojson.Unmarshal(v.Response, msg); err != nil {
			return nil, err
		}
		stream = append(stream, msg)

	default:
		return nil, errors.New("Unary method did not return result")
	}

	return stream, nil
}
