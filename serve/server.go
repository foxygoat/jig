// Package serve implements the "jig serve" command, serving GRPC services
// defined in a protoset file using the jsonnet contained in a method directory.
package serve

import (
	"errors"
	"net"
	"net/http"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"foxygo.at/jig/reflection"
)

type Server struct {
	Listen    string
	MethodDir string
	ProtoSet  string

	methods map[string]method
	gs      *grpc.Server
	lis     net.Listener
	files   *protoregistry.Files
}

var errUnknownHandler = errors.New("Unknown handler")

func (s *Server) setup() error {
	err := s.loadMethods()
	if err != nil {
		return err
	}

	s.gs = grpc.NewServer(
		grpc.StreamInterceptor(s.intercept),
		grpc.UnknownServiceHandler(unknownHandler),
	)

	reflection.NewService(s.files).Register(s.gs)

	s.lis, err = net.Listen("tcp", s.Listen)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) Run() error {
	if err := s.setup(); err != nil {
		return err
	}
	return http.Serve(s.lis, s)
}

func (s *Server) Stop() {
	s.gs.GracefulStop()
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, method := range s.methods {
		if rule, vars := method.matchHTTPRequest(r); rule != nil {
			method.serveHTTP(rule, vars, w, r)
			return
		}
	}
	s.gs.ServeHTTP(w, r)
}

func (s *Server) loadMethods() error {
	b, err := os.ReadFile(s.ProtoSet)
	if err != nil {
		return err
	}

	fds := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(b, fds); err != nil {
		return err
	}

	s.files, err = protodesc.NewFiles(fds)
	if err != nil {
		return err
	}

	s.methods = make(map[string]method)
	s.files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		sds := fd.Services()
		for i := 0; i < sds.Len(); i++ {
			mds := sds.Get(i).Methods()
			for j := 0; j < mds.Len(); j++ {
				m := newMethod(mds.Get(j), s.MethodDir)
				s.methods[m.fullMethod()] = m
			}
		}
		return true
	})
	return err
}

func (s *Server) intercept(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	// If the handler returns anything except errUnknownHandler, then we
	// have intercepted a real method and we are done now. Otherwise we
	// dispatch the method to a jsonnet handler.
	if err := handler(srv, ss); !errors.Is(err, errUnknownHandler) {
		return err
	}

	method, ok := s.methods[info.FullMethod]
	if !ok {
		return status.Errorf(codes.Unimplemented, "method not found: %s", info.FullMethod)
	}

	return method.call(ss)
}

// unknownHandler returns a sentinel error so the interceptor knows when
// calling it that is intercepting an unknown method and should dispatch
// it to jsonnet.
func unknownHandler(_ interface{}, stream grpc.ServerStream) error {
	return errUnknownHandler
}

type TestServer struct {
	Server
}

func (s *TestServer) Start() error {
	if err := s.setup(); err != nil {
		return err
	}
	go s.gs.Serve(s.lis) //nolint:errcheck
	return nil
}

func (s *TestServer) Addr() string {
	return s.lis.Addr().String()
}
