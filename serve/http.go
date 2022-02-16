package serve

import (
	context "context"
	"fmt"
	"mime"
	"net/http"
	"sync"

	"foxygo.at/jig/serve/httprule"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type httpMethod struct {
	desc protoreflect.MethodDescriptor
	rule *annotations.HttpRule
}

// Serve a google.api.http annotated method as HTTP
func (s *Server) serveHTTPMethod(m *httpMethod, vars map[string]string, w http.ResponseWriter, r *http.Request) {
	// TODO: Handle streaming calls.
	if err := s.callMethod(m.desc, &serverStream{
		req:        r,
		respWriter: w,
		rule:       m.rule,
		vars:       vars,
	}); err != nil {
		// TODO: Translate gRPC response codes.
		w.WriteHeader(500)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
}

func (s *Server) loadHTTPRules() error {
	s.files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		sds := fd.Services()
		for i := 0; i < sds.Len(); i++ {
			mds := sds.Get(i).Methods()
			for j := 0; j < mds.Len(); j++ {
				md := mds.Get(j)
				rules := httprule.Collect(md)
				for _, r := range rules {
					m := &httpMethod{desc: md, rule: r}
					s.httpMethods = append(s.httpMethods, m)
				}
			}
		}
		return true
	})
	return nil
}

type serverStream struct {
	mu         sync.Mutex
	once       sync.Once
	header     metadata.MD
	trailer    metadata.MD
	req        *http.Request
	respWriter http.ResponseWriter
	rule       *annotations.HttpRule
	vars       map[string]string
}

func (s *serverStream) SetHeader(md metadata.MD) error {
	if md.Len() == 0 {
		return nil
	}

	s.mu.Lock()
	s.header = metadata.Join(s.header, md)
	s.mu.Unlock()
	return nil
}

func (s *serverStream) SendHeader(md metadata.MD) error {
	return s.SetHeader(md)
}

func (s *serverStream) SetTrailer(md metadata.MD) {
	if md.Len() == 0 {
		return
	}

	s.mu.Lock()
	s.trailer = metadata.Join(s.trailer, md)
	s.mu.Unlock()
	return
}

func (s *serverStream) Context() context.Context {
	return s.req.Context()
}

func (s *serverStream) SendMsg(m interface{}) error {
	s.once.Do(func() {
		// TODO: Send headers
	})

	mediaType := httprule.ContentTypeJSON
	var err error
	accept := s.req.Header.Get("Accept")
	if accept != "" {
		mediaType, _, err = mime.ParseMediaType(accept)
		if err != nil {
			return err
		}
	}
	var marshal func(m proto.Message) ([]byte, error)
	switch mediaType {
	case httprule.ContentTypeBinaryProto:
		marshal = proto.Marshal
	case httprule.ContentTypeJSON:
		marshal = protojson.Marshal
	default:
		return fmt.Errorf("invalid content type %s", accept)
	}

	buf, err := marshal(m.(*dynamicpb.Message))
	if err != nil {
		return err
	}
	_, err = s.respWriter.Write(buf)
	return err
}

func (s *serverStream) RecvMsg(m interface{}) error {
	pb := m.(*dynamicpb.Message)
	return httprule.DecodeRequest(s.rule, s.vars, s.req, pb)
}

var _ grpc.ServerStream = &serverStream{}
