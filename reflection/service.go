package reflection

import (
	"io"

	"foxygo.at/protog/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	pb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Service is an implementation of the gRPC ServerReflection service, using a
// protoregistry.Files as the source of data for serving the reflection
// methods. No global protobuf/grpc state is consulted for serving.
type Service struct {
	registry *registry.Files
}

// FileDescriptorRanger iterates over a set of FileDescriptors.
type FileDescriptorRanger interface {
	// RangeFiles calls fn for each FileDescriptor in the Ranger, stopping
	// when there are no more or when fn returns false.
	RangeFiles(fn func(protoreflect.FileDescriptor) bool)
}

// NewService returns a new Service that implements the gRPC ServerReflection
// service from the given files registry. The files registry is cloned so that
// the reflection file descriptor can be registered without mutating the
// argument.
func NewService(files FileDescriptorRanger) *Service {
	r := cloneRegistry(files)
	// Ignore the RegisterFile error on the assumption it means the reflection
	// protofile is already registered.
	_ = r.RegisterFile(pb.File_reflection_grpc_reflection_v1alpha_reflection_proto)
	return &Service{registry: r}
}

// Register the s Service with the gs grpc ServiceRegistrar. This is a convenience
// function so the caller does not need to import the grpc_reflection_v1alpha1
// package.
func (s *Service) Register(gs grpc.ServiceRegistrar) {
	pb.RegisterServerReflectionServer(gs, s)
}

type streamHandler struct {
	registry *registry.Files
	seenFDs  map[string]bool
}

// ServerReflectionInfo implements pb.ServerReflectionServer
func (s *Service) ServerReflectionInfo(stream pb.ServerReflection_ServerReflectionInfoServer) error {
	sh := streamHandler{
		registry: s.registry,
		seenFDs:  make(map[string]bool),
	}
	return sh.handle(stream)
}

func (sh *streamHandler) handle(stream pb.ServerReflection_ServerReflectionInfoServer) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		out := &pb.ServerReflectionResponse{
			ValidHost:       in.Host,
			OriginalRequest: in,
		}

		switch req := in.MessageRequest.(type) {
		case *pb.ServerReflectionRequest_FileByFilename:
			out.MessageResponse, err = sh.fileByFilename(req)
		case *pb.ServerReflectionRequest_FileContainingSymbol:
			out.MessageResponse, err = sh.fileContainingSymbol(req)
		case *pb.ServerReflectionRequest_FileContainingExtension:
			out.MessageResponse, err = sh.fileContainingExtension(req)
		case *pb.ServerReflectionRequest_AllExtensionNumbersOfType:
			out.MessageResponse = sh.allExtensionNumbersOfType(req)
		case *pb.ServerReflectionRequest_ListServices:
			out.MessageResponse = sh.listServices(req)
		}

		if err != nil {
			out.MessageResponse = &pb.ServerReflectionResponse_ErrorResponse{
				ErrorResponse: &pb.ErrorResponse{
					ErrorCode:    int32(codes.NotFound),
					ErrorMessage: err.Error(),
				},
			}
		}
		if err := stream.Send(out); err != nil {
			return err
		}
	}
}

func (sh *streamHandler) fileByFilename(req *pb.ServerReflectionRequest_FileByFilename) (*pb.ServerReflectionResponse_FileDescriptorResponse, error) {
	fd, err := sh.registry.FindFileByPath(req.FileByFilename)
	if err != nil {
		return nil, err
	}
	return fileDescriptorResponse(sh.withDeps(fd))
}

func (sh *streamHandler) fileContainingSymbol(req *pb.ServerReflectionRequest_FileContainingSymbol) (*pb.ServerReflectionResponse_FileDescriptorResponse, error) {
	symbol := protoreflect.FullName(req.FileContainingSymbol)
	desc, err := sh.registry.FindDescriptorByName(symbol)
	if err != nil {
		return nil, err
	}
	return fileDescriptorResponse(sh.withDeps(desc.ParentFile()))
}

func (sh *streamHandler) fileContainingExtension(req *pb.ServerReflectionRequest_FileContainingExtension) (*pb.ServerReflectionResponse_FileDescriptorResponse, error) {
	message := protoreflect.FullName(req.FileContainingExtension.ContainingType)
	number := protoreflect.FieldNumber(req.FileContainingExtension.ExtensionNumber)
	et, err := sh.registry.FindExtensionByNumber(message, number)
	if err != nil {
		return nil, err
	}
	return fileDescriptorResponse(sh.withDeps(et.TypeDescriptor().ParentFile()))
}

func fileDescriptorResponse(fds []protoreflect.FileDescriptor) (*pb.ServerReflectionResponse_FileDescriptorResponse, error) {
	bs := make([][]byte, len(fds))
	var err error
	for i, fd := range fds {
		bs[i], err = proto.Marshal(protodesc.ToFileDescriptorProto(fd))
		if err != nil {
			return nil, err
		}
	}
	return &pb.ServerReflectionResponse_FileDescriptorResponse{
		FileDescriptorResponse: &pb.FileDescriptorResponse{FileDescriptorProto: bs},
	}, nil
}

func (sh *streamHandler) allExtensionNumbersOfType(req *pb.ServerReflectionRequest_AllExtensionNumbersOfType) *pb.ServerReflectionResponse_AllExtensionNumbersResponse {
	message := protoreflect.FullName(req.AllExtensionNumbersOfType)
	ets := sh.registry.GetExtensionsOfMessage(message)
	extNums := make([]int32, len(ets))
	for i, et := range ets {
		extNums[i] = int32(et.TypeDescriptor().Number())
	}

	return &pb.ServerReflectionResponse_AllExtensionNumbersResponse{
		AllExtensionNumbersResponse: &pb.ExtensionNumberResponse{
			BaseTypeName:    req.AllExtensionNumbersOfType,
			ExtensionNumber: extNums,
		},
	}
}

func (sh *streamHandler) listServices(req *pb.ServerReflectionRequest_ListServices) *pb.ServerReflectionResponse_ListServicesResponse {
	serviceResponses := []*pb.ServiceResponse{}
	sh.registry.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		sds := fd.Services()
		for i := 0; i < sds.Len(); i++ {
			sr := &pb.ServiceResponse{Name: string(sds.Get(i).FullName())}
			serviceResponses = append(serviceResponses, sr)
		}
		return true
	})

	return &pb.ServerReflectionResponse_ListServicesResponse{
		ListServicesResponse: &pb.ListServiceResponse{
			Service: serviceResponses,
		},
	}
}

func cloneRegistry(files FileDescriptorRanger) *registry.Files {
	clone := new(registry.Files)
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if err := clone.RegisterFile(fd); err != nil {
			panic(err)
		}
		return true
	})
	return clone
}

func (sh *streamHandler) withDeps(fd protoreflect.FileDescriptor) []protoreflect.FileDescriptor {
	fds := withDeps(fd, sh.seenFDs)
	// Always return the fd requested even if it has been seen before
	if len(fds) == 0 {
		fds = append(fds, fd)
	}
	return fds
}

func withDeps(fd protoreflect.FileDescriptor, seen map[string]bool) []protoreflect.FileDescriptor {
	result := []protoreflect.FileDescriptor{}
	if !seen[fd.Path()] {
		result = append(result, fd)
		seen[fd.Path()] = true
	}
	for i := 0; i < fd.Imports().Len(); i++ {
		result = append(result, withDeps(fd.Imports().Get(i), seen)...)
	}
	return result
}
