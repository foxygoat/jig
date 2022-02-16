package httprule

import (
	context "context"
	"fmt"
	"mime"
	"net/http"
	"sync"

	"foxygo.at/jig/registry"
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

type HandleForwardedGRPCRequest func(md protoreflect.MethodDescriptor, ss grpc.ServerStream) error

// Server serves protobuf methods, annotated using httprule options, over HTTP.
type Server struct {
	httpMethods []*httpMethod
	grpcHandler HandleForwardedGRPCRequest
}

func NewServer(files *registry.Files, handler HandleForwardedGRPCRequest) *Server {
	return &Server{
		httpMethods: loadHTTPRules(files),
		grpcHandler: handler,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, method := range s.httpMethods {
		if vars := MatchRequest(method.rule, r); vars != nil {
			s.serveHTTPMethod(method, vars, w, r)
			return
		}
	}
}

// Serve a google.api.http annotated method as HTTP
func (s *Server) serveHTTPMethod(m *httpMethod, vars map[string]string, w http.ResponseWriter, r *http.Request) {
	// TODO: Handle streaming calls.
	if err := s.grpcHandler(m.desc, &serverStream{
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

func loadHTTPRules(files *registry.Files) []*httpMethod {
	var httpMethods []*httpMethod
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		sds := fd.Services()
		for i := 0; i < sds.Len(); i++ {
			mds := sds.Get(i).Methods()
			for j := 0; j < mds.Len(); j++ {
				md := mds.Get(j)
				rules := Collect(md)
				for _, r := range rules {
					m := &httpMethod{desc: md, rule: r}
					httpMethods = append(httpMethods, m)
				}
			}
		}
		return true
	})
	return httpMethods
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

	mediaType := ContentTypeJSON
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
	case ContentTypeBinaryProto:
		marshal = proto.Marshal
	case ContentTypeJSON:
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
	return DecodeRequest(s.rule, s.vars, s.req, pb)
}

var _ grpc.ServerStream = &serverStream{}
