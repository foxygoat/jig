// Sample protos for exemplar testing

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v3.20.3
// source: exemplar/exemplar.proto

package exemplar

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	Exemplar_Sample_FullMethodName    = "/exemplar.Exemplar/Sample"
	Exemplar_WellKnown_FullMethodName = "/exemplar.Exemplar/WellKnown"
)

// ExemplarClient is the client API for Exemplar service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ExemplarClient interface {
	Sample(ctx context.Context, in *SampleRequest, opts ...grpc.CallOption) (*SampleResponse, error)
	WellKnown(ctx context.Context, in *SampleRequest, opts ...grpc.CallOption) (*WellKnownSample, error)
}

type exemplarClient struct {
	cc grpc.ClientConnInterface
}

func NewExemplarClient(cc grpc.ClientConnInterface) ExemplarClient {
	return &exemplarClient{cc}
}

func (c *exemplarClient) Sample(ctx context.Context, in *SampleRequest, opts ...grpc.CallOption) (*SampleResponse, error) {
	out := new(SampleResponse)
	err := c.cc.Invoke(ctx, Exemplar_Sample_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *exemplarClient) WellKnown(ctx context.Context, in *SampleRequest, opts ...grpc.CallOption) (*WellKnownSample, error) {
	out := new(WellKnownSample)
	err := c.cc.Invoke(ctx, Exemplar_WellKnown_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ExemplarServer is the server API for Exemplar service.
// All implementations must embed UnimplementedExemplarServer
// for forward compatibility
type ExemplarServer interface {
	Sample(context.Context, *SampleRequest) (*SampleResponse, error)
	WellKnown(context.Context, *SampleRequest) (*WellKnownSample, error)
	mustEmbedUnimplementedExemplarServer()
}

// UnimplementedExemplarServer must be embedded to have forward compatible implementations.
type UnimplementedExemplarServer struct {
}

func (UnimplementedExemplarServer) Sample(context.Context, *SampleRequest) (*SampleResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Sample not implemented")
}
func (UnimplementedExemplarServer) WellKnown(context.Context, *SampleRequest) (*WellKnownSample, error) {
	return nil, status.Errorf(codes.Unimplemented, "method WellKnown not implemented")
}
func (UnimplementedExemplarServer) mustEmbedUnimplementedExemplarServer() {}

// UnsafeExemplarServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ExemplarServer will
// result in compilation errors.
type UnsafeExemplarServer interface {
	mustEmbedUnimplementedExemplarServer()
}

func RegisterExemplarServer(s grpc.ServiceRegistrar, srv ExemplarServer) {
	s.RegisterService(&Exemplar_ServiceDesc, srv)
}

func _Exemplar_Sample_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SampleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExemplarServer).Sample(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Exemplar_Sample_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExemplarServer).Sample(ctx, req.(*SampleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Exemplar_WellKnown_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SampleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExemplarServer).WellKnown(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Exemplar_WellKnown_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExemplarServer).WellKnown(ctx, req.(*SampleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Exemplar_ServiceDesc is the grpc.ServiceDesc for Exemplar service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Exemplar_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "exemplar.Exemplar",
	HandlerType: (*ExemplarServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Sample",
			Handler:    _Exemplar_Sample_Handler,
		},
		{
			MethodName: "WellKnown",
			Handler:    _Exemplar_WellKnown_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "exemplar/exemplar.proto",
}
