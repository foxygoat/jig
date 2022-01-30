// Package serve implements the "jig serve" command, serving GRPC
// services via an evaluator.
package serve

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"strings"

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

func WithProtosets(protosets ...string) Option {
	return func(s *Server) error {
		s.protosets = append(s.protosets, protosets...)
		return nil
	}
}

func WithLogger(logger Logger) Option {
	return func(s *Server) error {
		s.log = logger
		return nil
	}
}

type Server struct {
	log       Logger
	gs        *grpc.Server
	files     *protoregistry.Files
	fs        fs.FS
	protosets []string
	eval      Evaluator
}

var errUnknownHandler = errors.New("Unknown handler")

// NewServer creates a new Server for given evaluator, e.g. Jsonnet and
// data Directories.
func NewServer(eval Evaluator, vfs fs.FS, options ...Option) (*Server, error) {
	s := &Server{
		files: new(protoregistry.Files),
		log:   NewLogger(os.Stderr, LogLevelError),
		eval:  eval,
		fs:    vfs,
	}
	for _, opt := range options {
		if err := opt(s); err != nil {
			return nil, err
		}
	}
	if err := s.loadProtosets(); err != nil {
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

func (s *Server) loadProtosets() error {
	seen := map[string]bool{}
	for _, protoset := range s.protosets {
		s.log.Debugf("loading protoset file: %s", protoset)
		b, err := os.ReadFile(protoset)
		if err != nil {
			return err
		}
		if err := s.addFiles(b, seen); err != nil {
			return err
		}
	}

	matches, err := fs.Glob(s.fs, "*.pb")
	if err != nil {
		return err
	}
	for _, match := range matches {
		if strings.HasPrefix(match, "_") {
			continue
		}
		b, err := fs.ReadFile(s.fs, match)
		if err != nil {
			return err
		}
		if err := s.addFiles(b, seen); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) addFiles(b []byte, seen map[string]bool) error {
	fds := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(b, fds); err != nil {
		return err
	}
	files, err := protodesc.NewFiles(fds)
	if err != nil {
		return err
	}
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if seen[fd.Path()] {
			return true
		}
		seen[fd.Path()] = true
		s.log.Debugf("loading file descriptor %s", fd.Path())
		err := s.files.RegisterFile(fd)
		if err != nil {
			s.log.Errorf("cannot register %q: %v", fd.FullName(), err)
		}
		return true
	})
	return nil
}

func (s *Server) lookupMethod(name protoreflect.FullName) protoreflect.MethodDescriptor {
	desc, err := s.files.FindDescriptorByName(name)
	if err != nil {
		return nil
	}
	md, ok := desc.(protoreflect.MethodDescriptor)
	if !ok {
		return nil
	}
	return md
}

func (s *Server) intercept(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	s.log.Debugf("%s: new request", info.FullMethod)
	// If the handler returns anything except errUnknownHandler, then we
	// have intercepted a real method and we are done now. Otherwise we
	// dispatch the method to the evaluator.
	if err := handler(srv, ss); !errors.Is(err, errUnknownHandler) {
		if err != nil {
			s.log.Errorf("%s: %s", info.FullMethod, err)
		}
		return err
	}

	// Convert /pkg.service/method -> pkg.service.method
	fullMethod := protoreflect.FullName(strings.ReplaceAll(info.FullMethod[1:], "/", "."))
	md := s.lookupMethod(fullMethod)
	if md == nil {
		s.log.Warnf("%s: method not found", fullMethod)
		return status.Errorf(codes.Unimplemented, "method not found: %s", fullMethod)
	}

	err := s.callMethod(md, ss)
	if err != nil {
		s.log.Errorf("%s: %s", info.FullMethod, err)
	}
	return err
}

// unknownHandler returns a sentinel error so the interceptor knows when
// calling it that is intercepting an unknown method and should dispatch
// it to the evaluator.
func unknownHandler(_ interface{}, stream grpc.ServerStream) error {
	return errUnknownHandler
}

type TestServer struct {
	Server
	lis net.Listener
}

// NewTestServer starts and returns a new TestServer.
// The caller should call Stop when finished, to shut it down.
func NewTestServer(eval Evaluator, vfs fs.FS, options ...Option) *TestServer {
	s, err := NewServer(eval, vfs, options...)
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
