// Package serve implements the "jig serve" command, serving GRPC services
// defined in a protoset file using the jsonnet contained in a method directory.
package serve

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"

	"foxygo.at/jig/reflection"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Option is a functional option to configure Server
type Option func(s *Server) error

func WithFS(fs fs.FS) Option {
	return func(s *Server) error {
		s.fs = fs
		return nil
	}
}

type Server struct {
	methodDir string
	protoSet  string

	methods map[string]method
	gs      *grpc.Server
	files   *protoregistry.Files
	fs      fs.FS
}

var errUnknownHandler = errors.New("Unknown handler")

// NewServer creates a new Server. Its API is currently unstable.
func NewServer(methodDir, protoSet string, options ...Option) (*Server, error) {
	s := &Server{
		methodDir: methodDir,
		protoSet:  protoSet,
	}
	for _, opt := range options {
		if err := opt(s); err != nil {
			return nil, err
		}
	}
	if err := s.loadMethods(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Server) Serve(lis net.Listener) error {
	s.gs = grpc.NewServer(
		grpc.StreamInterceptor(s.intercept),
		grpc.UnknownServiceHandler(unknownHandler),
	)
	reflection.NewService(s.files).Register(s.gs)
	return s.gs.Serve(lis)
}

func (s *Server) ListenAndServe(listenAddr string) error {
	l, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	return s.Serve(l)
}

func (s *Server) Stop() {
	if s.gs != nil {
		s.gs.GracefulStop()
	}
}

func (s *Server) loadMethods() error {
	var b []byte
	var err error
	methodFS := s.fs
	if s.fs != nil {
		b, err = fs.ReadFile(s.fs, s.protoSet)
	} else {
		b, err = os.ReadFile(s.protoSet)
		methodFS = os.DirFS(s.methodDir)
	}
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
				m := newMethod(mds.Get(j), methodFS)
				s.methods[m.fullMethod()] = m
			}
		}
		return true
	})
	return nil
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
	lis net.Listener
}

// NewTestServer starts and returns a new TestServer.
// The caller should call Stop when finished, to shut it down.
func NewTestServer(methodDir, protoSet string, options ...Option) *TestServer {
	s, err := NewServer(methodDir, protoSet, options...)
	if err != nil {
		panic(fmt.Sprintf("failed to create TestServer: %v", err))
	}
	ts := &TestServer{Server: *s}
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(fmt.Sprintf("TestServer failed to listen: %v", err))
	}
	ts.lis = l
	go ts.Serve(l) //nolint: errcheck
	return ts
}

func (ts *TestServer) Addr() string {
	return ts.lis.Addr().String()
}
