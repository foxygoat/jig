package serve

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/google/go-jsonnet"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	if m.desc.IsStreamingClient() || m.desc.IsStreamingServer() {
		return status.Errorf(codes.Unimplemented, "streaming not supported: %s", m.desc.FullName())
	}

	req := dynamicpb.NewMessage(m.desc.Input())
	if err := ss.RecvMsg(req); err != nil {
		return err
	}
	inputJSON, err := makeInputJSON(req)
	if err != nil {
		return err
	}

	vm := jsonnet.MakeVM()
	// TODO(camh): Add jsonnext.Importer
	vm.TLACode("input", inputJSON)
	outputJSON, err := vm.EvaluateFile(m.filename)
	if err != nil {
		return err
	}

	resp := dynamicpb.NewMessage(m.desc.Output())
	if err := parseOutputJSON(outputJSON, resp); err != nil {
		return err
	}
	if err := ss.SendMsg(resp); err != nil {
		return err
	}
	return nil
}

func makeInputJSON(msg *dynamicpb.Message) (string, error) {
	mo := protojson.MarshalOptions{EmitUnpopulated: true}
	b, err := mo.Marshal(msg)
	if err != nil {
		return "", err
	}
	v := struct {
		Request json.RawMessage `json:"request"`
	}{Request: b}
	input, err := json.Marshal(&v)
	if err != nil {
		return "", err
	}

	return string(input), nil
}

func parseOutputJSON(output string, msg *dynamicpb.Message) error {
	v := struct {
		Response json.RawMessage `json:"response"`
	}{}
	if err := json.Unmarshal([]byte(output), &v); err != nil {
		return err
	}

	if err := protojson.Unmarshal(v.Response, msg); err != nil {
		return err
	}

	return nil
}
