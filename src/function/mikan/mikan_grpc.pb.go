// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package mikan

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

// MikanClient is the client API for Mikan service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type MikanClient interface {
	GetInfo(ctx context.Context, in *Param, opts ...grpc.CallOption) (*Info, error)
}

type mikanClient struct {
	cc grpc.ClientConnInterface
}

func NewMikanClient(cc grpc.ClientConnInterface) MikanClient {
	return &mikanClient{cc}
}

func (c *mikanClient) GetInfo(ctx context.Context, in *Param, opts ...grpc.CallOption) (*Info, error) {
	out := new(Info)
	err := c.cc.Invoke(ctx, "/mikan.Mikan/GetInfo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MikanServer is the server API for Mikan service.
// All implementations must embed UnimplementedMikanServer
// for forward compatibility
type MikanServer interface {
	GetInfo(context.Context, *Param) (*Info, error)
	mustEmbedUnimplementedMikanServer()
}

// UnimplementedMikanServer must be embedded to have forward compatible implementations.
type UnimplementedMikanServer struct {
}

func (UnimplementedMikanServer) GetInfo(context.Context, *Param) (*Info, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetInfo not implemented")
}
func (UnimplementedMikanServer) mustEmbedUnimplementedMikanServer() {}

// UnsafeMikanServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to MikanServer will
// result in compilation errors.
type UnsafeMikanServer interface {
	mustEmbedUnimplementedMikanServer()
}

func RegisterMikanServer(s grpc.ServiceRegistrar, srv MikanServer) {
	s.RegisterService(&Mikan_ServiceDesc, srv)
}

func _Mikan_GetInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Param)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MikanServer).GetInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/mikan.Mikan/GetInfo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MikanServer).GetInfo(ctx, req.(*Param))
	}
	return interceptor(ctx, in, info, handler)
}

// Mikan_ServiceDesc is the grpc.ServiceDesc for Mikan service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Mikan_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "mikan.Mikan",
	HandlerType: (*MikanServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetInfo",
			Handler:    _Mikan_GetInfo_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "mikan.proto",
}