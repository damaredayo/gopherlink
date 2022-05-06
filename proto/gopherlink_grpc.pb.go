// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.19.4
// source: gopherlink.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// GopherlinkClient is the client API for Gopherlink service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type GopherlinkClient interface {
	GetStatusStream(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (Gopherlink_GetStatusStreamClient, error)
	CreatePlayer(ctx context.Context, in *DiscordVoiceServer, opts ...grpc.CallOption) (*PlayerResponse, error)
	AddSong(ctx context.Context, in *SongRequest, opts ...grpc.CallOption) (*SongAdded, error)
	PauseSong(ctx context.Context, in *SongPauseRequest, opts ...grpc.CallOption) (*SongInfo, error)
	RemoveSong(ctx context.Context, in *SongRequest, opts ...grpc.CallOption) (*SongRemoved, error)
	NowPlaying(ctx context.Context, in *NowPlayingRequest, opts ...grpc.CallOption) (*SongInfo, error)
	GetQueue(ctx context.Context, in *QueueRequest, opts ...grpc.CallOption) (*Queue, error)
	Seek(ctx context.Context, in *SeekRequest, opts ...grpc.CallOption) (*SongInfo, error)
}

type gopherlinkClient struct {
	cc grpc.ClientConnInterface
}

func NewGopherlinkClient(cc grpc.ClientConnInterface) GopherlinkClient {
	return &gopherlinkClient{cc}
}

func (c *gopherlinkClient) GetStatusStream(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (Gopherlink_GetStatusStreamClient, error) {
	stream, err := c.cc.NewStream(ctx, &Gopherlink_ServiceDesc.Streams[0], "/database.Gopherlink/GetStatusStream", opts...)
	if err != nil {
		return nil, err
	}
	x := &gopherlinkGetStatusStreamClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Gopherlink_GetStatusStreamClient interface {
	Recv() (*Status, error)
	grpc.ClientStream
}

type gopherlinkGetStatusStreamClient struct {
	grpc.ClientStream
}

func (x *gopherlinkGetStatusStreamClient) Recv() (*Status, error) {
	m := new(Status)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *gopherlinkClient) CreatePlayer(ctx context.Context, in *DiscordVoiceServer, opts ...grpc.CallOption) (*PlayerResponse, error) {
	out := new(PlayerResponse)
	err := c.cc.Invoke(ctx, "/database.Gopherlink/CreatePlayer", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gopherlinkClient) AddSong(ctx context.Context, in *SongRequest, opts ...grpc.CallOption) (*SongAdded, error) {
	out := new(SongAdded)
	err := c.cc.Invoke(ctx, "/database.Gopherlink/AddSong", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gopherlinkClient) PauseSong(ctx context.Context, in *SongPauseRequest, opts ...grpc.CallOption) (*SongInfo, error) {
	out := new(SongInfo)
	err := c.cc.Invoke(ctx, "/database.Gopherlink/PauseSong", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gopherlinkClient) RemoveSong(ctx context.Context, in *SongRequest, opts ...grpc.CallOption) (*SongRemoved, error) {
	out := new(SongRemoved)
	err := c.cc.Invoke(ctx, "/database.Gopherlink/RemoveSong", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gopherlinkClient) NowPlaying(ctx context.Context, in *NowPlayingRequest, opts ...grpc.CallOption) (*SongInfo, error) {
	out := new(SongInfo)
	err := c.cc.Invoke(ctx, "/database.Gopherlink/NowPlaying", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gopherlinkClient) GetQueue(ctx context.Context, in *QueueRequest, opts ...grpc.CallOption) (*Queue, error) {
	out := new(Queue)
	err := c.cc.Invoke(ctx, "/database.Gopherlink/GetQueue", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gopherlinkClient) Seek(ctx context.Context, in *SeekRequest, opts ...grpc.CallOption) (*SongInfo, error) {
	out := new(SongInfo)
	err := c.cc.Invoke(ctx, "/database.Gopherlink/Seek", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GopherlinkServer is the server API for Gopherlink service.
// All implementations must embed UnimplementedGopherlinkServer
// for forward compatibility
type GopherlinkServer interface {
	GetStatusStream(*emptypb.Empty, Gopherlink_GetStatusStreamServer) error
	CreatePlayer(context.Context, *DiscordVoiceServer) (*PlayerResponse, error)
	AddSong(context.Context, *SongRequest) (*SongAdded, error)
	PauseSong(context.Context, *SongPauseRequest) (*SongInfo, error)
	RemoveSong(context.Context, *SongRequest) (*SongRemoved, error)
	NowPlaying(context.Context, *NowPlayingRequest) (*SongInfo, error)
	GetQueue(context.Context, *QueueRequest) (*Queue, error)
	Seek(context.Context, *SeekRequest) (*SongInfo, error)
	mustEmbedUnimplementedGopherlinkServer()
}

// UnimplementedGopherlinkServer must be embedded to have forward compatible implementations.
type UnimplementedGopherlinkServer struct {
}

func (UnimplementedGopherlinkServer) GetStatusStream(*emptypb.Empty, Gopherlink_GetStatusStreamServer) error {
	return status.Errorf(codes.Unimplemented, "method GetStatusStream not implemented")
}
func (UnimplementedGopherlinkServer) CreatePlayer(context.Context, *DiscordVoiceServer) (*PlayerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreatePlayer not implemented")
}
func (UnimplementedGopherlinkServer) AddSong(context.Context, *SongRequest) (*SongAdded, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddSong not implemented")
}
func (UnimplementedGopherlinkServer) PauseSong(context.Context, *SongPauseRequest) (*SongInfo, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PauseSong not implemented")
}
func (UnimplementedGopherlinkServer) RemoveSong(context.Context, *SongRequest) (*SongRemoved, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveSong not implemented")
}
func (UnimplementedGopherlinkServer) NowPlaying(context.Context, *NowPlayingRequest) (*SongInfo, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NowPlaying not implemented")
}
func (UnimplementedGopherlinkServer) GetQueue(context.Context, *QueueRequest) (*Queue, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetQueue not implemented")
}
func (UnimplementedGopherlinkServer) Seek(context.Context, *SeekRequest) (*SongInfo, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Seek not implemented")
}
func (UnimplementedGopherlinkServer) mustEmbedUnimplementedGopherlinkServer() {}

// UnsafeGopherlinkServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to GopherlinkServer will
// result in compilation errors.
type UnsafeGopherlinkServer interface {
	mustEmbedUnimplementedGopherlinkServer()
}

func RegisterGopherlinkServer(s grpc.ServiceRegistrar, srv GopherlinkServer) {
	s.RegisterService(&Gopherlink_ServiceDesc, srv)
}

func _Gopherlink_GetStatusStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(emptypb.Empty)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(GopherlinkServer).GetStatusStream(m, &gopherlinkGetStatusStreamServer{stream})
}

type Gopherlink_GetStatusStreamServer interface {
	Send(*Status) error
	grpc.ServerStream
}

type gopherlinkGetStatusStreamServer struct {
	grpc.ServerStream
}

func (x *gopherlinkGetStatusStreamServer) Send(m *Status) error {
	return x.ServerStream.SendMsg(m)
}

func _Gopherlink_CreatePlayer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DiscordVoiceServer)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GopherlinkServer).CreatePlayer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/database.Gopherlink/CreatePlayer",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GopherlinkServer).CreatePlayer(ctx, req.(*DiscordVoiceServer))
	}
	return interceptor(ctx, in, info, handler)
}

func _Gopherlink_AddSong_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SongRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GopherlinkServer).AddSong(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/database.Gopherlink/AddSong",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GopherlinkServer).AddSong(ctx, req.(*SongRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Gopherlink_PauseSong_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SongPauseRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GopherlinkServer).PauseSong(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/database.Gopherlink/PauseSong",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GopherlinkServer).PauseSong(ctx, req.(*SongPauseRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Gopherlink_RemoveSong_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SongRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GopherlinkServer).RemoveSong(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/database.Gopherlink/RemoveSong",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GopherlinkServer).RemoveSong(ctx, req.(*SongRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Gopherlink_NowPlaying_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NowPlayingRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GopherlinkServer).NowPlaying(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/database.Gopherlink/NowPlaying",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GopherlinkServer).NowPlaying(ctx, req.(*NowPlayingRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Gopherlink_GetQueue_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueueRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GopherlinkServer).GetQueue(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/database.Gopherlink/GetQueue",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GopherlinkServer).GetQueue(ctx, req.(*QueueRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Gopherlink_Seek_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SeekRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GopherlinkServer).Seek(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/database.Gopherlink/Seek",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GopherlinkServer).Seek(ctx, req.(*SeekRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Gopherlink_ServiceDesc is the grpc.ServiceDesc for Gopherlink service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Gopherlink_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "database.Gopherlink",
	HandlerType: (*GopherlinkServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreatePlayer",
			Handler:    _Gopherlink_CreatePlayer_Handler,
		},
		{
			MethodName: "AddSong",
			Handler:    _Gopherlink_AddSong_Handler,
		},
		{
			MethodName: "PauseSong",
			Handler:    _Gopherlink_PauseSong_Handler,
		},
		{
			MethodName: "RemoveSong",
			Handler:    _Gopherlink_RemoveSong_Handler,
		},
		{
			MethodName: "NowPlaying",
			Handler:    _Gopherlink_NowPlaying_Handler,
		},
		{
			MethodName: "GetQueue",
			Handler:    _Gopherlink_GetQueue_Handler,
		},
		{
			MethodName: "Seek",
			Handler:    _Gopherlink_Seek_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "GetStatusStream",
			Handler:       _Gopherlink_GetStatusStream_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "gopherlink.proto",
}
