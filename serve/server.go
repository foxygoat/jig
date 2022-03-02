// Package serve implements the "jig serve" command, serving GRPC
// services via an evaluator.
package serve

import (
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"strings"

	"foxygo.at/jig/log"
	"foxygo.at/jig/reflection"
	"foxygo.at/jig/registry"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
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

func WithLogger(logger log.Logger) Option {
	return func(s *Server) error {
		s.log = logger
		return nil
	}
}

type Server struct {
	Files *registry.Files

	log       log.Logger
	gs        *grpc.Server
	http      http.Handler
	fs        fs.FS
	protosets []string
	eval      Evaluator
}

// NewServer creates a new Server for given evaluator, e.g. Jsonnet and
// data Directories.
func NewServer(eval Evaluator, vfs fs.FS, options ...Option) (*Server, error) {
	s := &Server{
		Files: new(registry.Files),
		log:   log.NewLogger(os.Stderr, log.LogLevelError),
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

// SetHTTPHandler sets a http.Handler to be called for non-grpc traffic.
// It must be called before Serve or ListenAndServe are called.
func (s *Server) SetHTTPHandler(handler http.Handler) {
	s.http = handler
}

func (s *Server) Serve(lis net.Listener) error {
	s.gs = grpc.NewServer(grpc.UnknownServiceHandler(s.UnknownHandler))
	reflection.NewService(&s.Files.Files).Register(s.gs)
	if s.http != nil {
		return http.Serve(lis, h2c.NewHandler(s, &http2.Server{}))
	}
	return s.gs.Serve(lis)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
		s.gs.ServeHTTP(w, r)
		return
	}
	s.http.ServeHTTP(w, r)
}

func (s *Server) ListenAndServe(listenAddr string) error {
	l, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	s.log.Infof("Listening on %s", listenAddr)
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
		s.log.Debugf("loading discovered protoset file: %s", match)
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
		err := s.Files.RegisterFile(fd)
		if err != nil {
			s.log.Errorf("cannot register %q: %v", fd.FullName(), err)
		}
		return true
	})
	return nil
}

func (s *Server) lookupMethod(name protoreflect.FullName) protoreflect.MethodDescriptor {
	desc, err := s.Files.FindDescriptorByName(name)
	if err != nil {
		return nil
	}
	md, ok := desc.(protoreflect.MethodDescriptor)
	if !ok {
		return nil
	}
	return md
}

// UnknownHandler handles gRPC method calls that are not statically
// implemented. It is registered as the UnknownServiceHandler of the grpc
// server.
//
// UnknownHandler can be called from a non-grpc.Server entry point, possibly a
// HTTP server or even an internal dispatch bypassing the network. In order for
// this handler to know which method is being invoked, the `srv` argument can
// be (ab)used to pass the full name of the method. In this case the dynamic
// type of `srv` must be `protoreflect.FullName` and it must contain the
// fully-qualified method name (pkg.service.method).
//
// Alternatively, the stream context (ss.Context()) must contain a
// grpc.ServerTransportStream, retrievable with
// grpc.ServerTransportStreamFromContext(), on which the `Method()` method will
// be called to find the method name. This should return the method as a HTTP
// path (/pkg.service/method), as is done by grpc.Server.
func (s *Server) UnknownHandler(srv interface{}, ss grpc.ServerStream) error {
	var fullMethod protoreflect.FullName
	var ok bool
	if fullMethod, ok = srv.(protoreflect.FullName); !ok {
		methodPath, ok := grpc.Method(ss.Context())
		if !ok {
			return status.Errorf(codes.Internal, "no method in stream context")
		}
		// Convert /pkg.service/method -> pkg.service.method
		fullMethod = protoreflect.FullName(strings.ReplaceAll(methodPath[1:], "/", "."))
	}

	s.log.Debugf("%s: new request", fullMethod)

	md := s.lookupMethod(fullMethod)
	if md == nil {
		s.log.Warnf("%s: method not found", fullMethod)
		return status.Errorf(codes.Unimplemented, "method not found: %s", fullMethod)
	}

	if err := s.callMethod(md, ss); err != nil {
		s.log.Errorf("%s: %s", fullMethod, err)
		return err
	}

	return nil
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
