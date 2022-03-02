package httprule

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"os"

	"foxygo.at/jig/log"
	"foxygo.at/protog/registry"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type httpMethod struct {
	desc protoreflect.MethodDescriptor
	rule *annotations.HttpRule
}

// Server serves protobuf methods, annotated using httprule options, over HTTP.
type Server struct {
	httpMethods []*httpMethod
	grpcHandler grpc.StreamHandler
	log         log.Logger
}

func NewServer(files *registry.Files, handler grpc.StreamHandler) *Server {
	return &Server{
		httpMethods: loadHTTPRules(files),
		grpcHandler: handler,
		log:         log.NewLogger(os.Stderr, log.LogLevelError),
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
	ss := &serverStream{
		req:        r,
		respWriter: w,
		rule:       m.rule,
		vars:       vars,
		log:        s.log,
	}
	if err := s.grpcHandler(m.desc.FullName(), ss); err != nil {
		ss.writeError(err)
		return
	}
	ss.writeResp()
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
	header     metadata.MD
	trailer    metadata.MD
	req        *http.Request
	respWriter http.ResponseWriter
	rule       *annotations.HttpRule
	vars       map[string]string
	acceptType string
	resp       proto.Message
	log        log.Logger
}

var _ grpc.ServerStream = &serverStream{}

func (s *serverStream) SetHeader(md metadata.MD) error {
	if md.Len() == 0 {
		return nil
	}

	s.header = metadata.Join(s.header, md)
	return nil
}

func (s *serverStream) SendHeader(md metadata.MD) error {
	return s.SetHeader(md)
}

func (s *serverStream) SetTrailer(md metadata.MD) {
	if md.Len() == 0 {
		return
	}

	s.trailer = metadata.Join(s.trailer, md)
}

func (s *serverStream) Context() context.Context {
	// TODO: Propagate metadata to headers.
	return s.req.Context()
}

func (s *serverStream) SendMsg(m interface{}) error {
	// Message is buffered until the RPC returns since we don't support client streaming... yet.
	if s.resp != nil {
		panic("only one response expected!")
	}
	s.resp = m.(proto.Message)
	return nil
}

func (s *serverStream) RecvMsg(m interface{}) error {
	var err error
	s.acceptType, err = getAcceptType(s.req)
	if err != nil {
		return err
	}

	pb := m.(*dynamicpb.Message)
	return DecodeRequest(s.rule, s.vars, s.req, pb)
}

func (s *serverStream) writeResp() {
	// TODO: forward headers and trailers.
	msg, err := marshalerForContentType(s.acceptType)(s.resp)
	if err != nil {
		s.writeError(err)
		return
	}
	if _, err = s.respWriter.Write(msg); err != nil {
		s.log.Errorf("failed to write response")
		return
	}
}

func (s *serverStream) writeError(err error) {
	// Fallback message if error marshalling fails.
	const errMarshalFailed = `{"code": 13, "message": "failed to marshal error message"}`

	w := s.respWriter
	st := status.Convert(err)
	// If we don't understand the "Accept" header, error back in JSON without setting Content-Type.
	marshaler := protojson.Marshal
	if s.acceptType != "" {
		marshaler = marshalerForContentType(s.acceptType)
		w.Header().Set("Content-Type", s.acceptType)
	}

	buf, err := marshaler(st.Proto())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if _, err = w.Write([]byte(errMarshalFailed)); err != nil {
			s.log.Errorf("failed to write error response: %+v", err)
		}
		return
	}
	s.respWriter.WriteHeader(HTTPStatusFromCode(st.Code()))
	if _, err = s.respWriter.Write(buf); err != nil {
		s.log.Errorf("failed to write error response: %+v", err)
	}
}

func getAcceptType(r *http.Request) (string, error) {
	var err error
	mediaType := ContentTypeJSON
	// TODO: There's a lot more to parsing Accept headers...
	accept := r.Header.Get("Accept")
	if accept != "" && accept != "*/*" {
		mediaType, _, err = mime.ParseMediaType(accept)
		if err != nil {
			return "", err
		}
	}
	if mediaType != ContentTypeBinaryProto && mediaType != ContentTypeJSON {
		return "", fmt.Errorf("invalid Accept content type %s", accept)
	}
	return mediaType, nil
}

func marshalerForContentType(mediaType string) func(m proto.Message) ([]byte, error) {
	switch mediaType {
	case ContentTypeBinaryProto:
		return proto.Marshal
	case ContentTypeJSON:
		return protojson.Marshal
	default:
		panic("invalid content type")
	}
}
