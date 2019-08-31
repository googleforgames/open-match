// Code generated by protoc-gen-go. DO NOT EDIT.
// source: internal/api/synchronizer.proto

package ipb

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger/options"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	math "math"
	pb "open-match.dev/open-match/pkg/pb"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type RegisterRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *RegisterRequest) Reset()         { *m = RegisterRequest{} }
func (m *RegisterRequest) String() string { return proto.CompactTextString(m) }
func (*RegisterRequest) ProtoMessage()    {}
func (*RegisterRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_35ff6b85fea1c4b7, []int{0}
}

func (m *RegisterRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RegisterRequest.Unmarshal(m, b)
}
func (m *RegisterRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RegisterRequest.Marshal(b, m, deterministic)
}
func (m *RegisterRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RegisterRequest.Merge(m, src)
}
func (m *RegisterRequest) XXX_Size() int {
	return xxx_messageInfo_RegisterRequest.Size(m)
}
func (m *RegisterRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_RegisterRequest.DiscardUnknown(m)
}

var xxx_messageInfo_RegisterRequest proto.InternalMessageInfo

type RegisterResponse struct {
	// Identifier for this request valid for the current synchronization cycle.
	Id                   string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *RegisterResponse) Reset()         { *m = RegisterResponse{} }
func (m *RegisterResponse) String() string { return proto.CompactTextString(m) }
func (*RegisterResponse) ProtoMessage()    {}
func (*RegisterResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_35ff6b85fea1c4b7, []int{1}
}

func (m *RegisterResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RegisterResponse.Unmarshal(m, b)
}
func (m *RegisterResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RegisterResponse.Marshal(b, m, deterministic)
}
func (m *RegisterResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RegisterResponse.Merge(m, src)
}
func (m *RegisterResponse) XXX_Size() int {
	return xxx_messageInfo_RegisterResponse.Size(m)
}
func (m *RegisterResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_RegisterResponse.DiscardUnknown(m)
}

var xxx_messageInfo_RegisterResponse proto.InternalMessageInfo

func (m *RegisterResponse) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

type EvaluateProposalsRequest struct {
	// List of proposals to evaluate in the current synchronization cycle.
	Match *pb.Match `protobuf:"bytes,1,opt,name=match,proto3" json:"match,omitempty"`
	// Identifier for this request issued during request registration.
	Id                   string   `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *EvaluateProposalsRequest) Reset()         { *m = EvaluateProposalsRequest{} }
func (m *EvaluateProposalsRequest) String() string { return proto.CompactTextString(m) }
func (*EvaluateProposalsRequest) ProtoMessage()    {}
func (*EvaluateProposalsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_35ff6b85fea1c4b7, []int{2}
}

func (m *EvaluateProposalsRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_EvaluateProposalsRequest.Unmarshal(m, b)
}
func (m *EvaluateProposalsRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_EvaluateProposalsRequest.Marshal(b, m, deterministic)
}
func (m *EvaluateProposalsRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EvaluateProposalsRequest.Merge(m, src)
}
func (m *EvaluateProposalsRequest) XXX_Size() int {
	return xxx_messageInfo_EvaluateProposalsRequest.Size(m)
}
func (m *EvaluateProposalsRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_EvaluateProposalsRequest.DiscardUnknown(m)
}

var xxx_messageInfo_EvaluateProposalsRequest proto.InternalMessageInfo

func (m *EvaluateProposalsRequest) GetMatch() *pb.Match {
	if m != nil {
		return m.Match
	}
	return nil
}

func (m *EvaluateProposalsRequest) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

type EvaluateProposalsResponse struct {
	// Results from evaluating proposals for this request.
	Match                *pb.Match `protobuf:"bytes,1,opt,name=match,proto3" json:"match,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *EvaluateProposalsResponse) Reset()         { *m = EvaluateProposalsResponse{} }
func (m *EvaluateProposalsResponse) String() string { return proto.CompactTextString(m) }
func (*EvaluateProposalsResponse) ProtoMessage()    {}
func (*EvaluateProposalsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_35ff6b85fea1c4b7, []int{3}
}

func (m *EvaluateProposalsResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_EvaluateProposalsResponse.Unmarshal(m, b)
}
func (m *EvaluateProposalsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_EvaluateProposalsResponse.Marshal(b, m, deterministic)
}
func (m *EvaluateProposalsResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EvaluateProposalsResponse.Merge(m, src)
}
func (m *EvaluateProposalsResponse) XXX_Size() int {
	return xxx_messageInfo_EvaluateProposalsResponse.Size(m)
}
func (m *EvaluateProposalsResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_EvaluateProposalsResponse.DiscardUnknown(m)
}

var xxx_messageInfo_EvaluateProposalsResponse proto.InternalMessageInfo

func (m *EvaluateProposalsResponse) GetMatch() *pb.Match {
	if m != nil {
		return m.Match
	}
	return nil
}

func init() {
	proto.RegisterType((*RegisterRequest)(nil), "api.internal.RegisterRequest")
	proto.RegisterType((*RegisterResponse)(nil), "api.internal.RegisterResponse")
	proto.RegisterType((*EvaluateProposalsRequest)(nil), "api.internal.EvaluateProposalsRequest")
	proto.RegisterType((*EvaluateProposalsResponse)(nil), "api.internal.EvaluateProposalsResponse")
}

func init() { proto.RegisterFile("internal/api/synchronizer.proto", fileDescriptor_35ff6b85fea1c4b7) }

var fileDescriptor_35ff6b85fea1c4b7 = []byte{
	// 550 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x84, 0x93, 0x4d, 0x6f, 0xd3, 0x4c,
	0x10, 0xc7, 0x65, 0xe7, 0x79, 0x0a, 0x2c, 0x15, 0xb4, 0x7b, 0x40, 0x69, 0x78, 0xdb, 0xba, 0x52,
	0x89, 0x22, 0xe2, 0x4d, 0x43, 0x4e, 0x41, 0x95, 0x5a, 0x20, 0x87, 0x4a, 0xe1, 0x45, 0xae, 0xc4,
	0x81, 0xdb, 0xc6, 0x1e, 0xec, 0x45, 0xf6, 0xee, 0xb2, 0xbb, 0x4e, 0x81, 0x23, 0x37, 0xae, 0x70,
	0x00, 0xf1, 0x41, 0xf8, 0x22, 0x9c, 0xb9, 0xf1, 0x41, 0x90, 0xed, 0xb8, 0x49, 0x9a, 0x56, 0x3d,
	0x59, 0xde, 0xf9, 0xcf, 0x7f, 0x66, 0x7e, 0xbb, 0x83, 0xee, 0x73, 0x61, 0x41, 0x0b, 0x96, 0x52,
	0xa6, 0x38, 0x35, 0x1f, 0x45, 0x98, 0x68, 0x29, 0xf8, 0x27, 0xd0, 0xbe, 0xd2, 0xd2, 0x4a, 0xbc,
	0xce, 0x14, 0xf7, 0x6b, 0x51, 0x0b, 0x17, 0xaa, 0x0c, 0x8c, 0x61, 0x31, 0x98, 0x4a, 0xd1, 0xba,
	0x13, 0x4b, 0x19, 0xa7, 0x50, 0x1a, 0x30, 0x21, 0xa4, 0x65, 0x96, 0x4b, 0x51, 0x47, 0x1f, 0x96,
	0x9f, 0xb0, 0x1b, 0x83, 0xe8, 0x9a, 0x13, 0x16, 0xc7, 0xa0, 0xa9, 0x54, 0xa5, 0x62, 0x55, 0xed,
	0x6d, 0xa2, 0x9b, 0x01, 0xc4, 0xdc, 0x58, 0xd0, 0x01, 0xbc, 0xcf, 0xc1, 0x58, 0xcf, 0x43, 0x1b,
	0xf3, 0x23, 0xa3, 0xa4, 0x30, 0x80, 0x6f, 0x20, 0x97, 0x47, 0x4d, 0x87, 0x38, 0xed, 0x6b, 0x81,
	0xcb, 0x23, 0x6f, 0x8c, 0x9a, 0xa3, 0x29, 0x4b, 0x73, 0x66, 0xe1, 0x95, 0x96, 0x4a, 0x1a, 0x96,
	0x9a, 0x59, 0x3e, 0x26, 0xe8, 0xff, 0x8c, 0xd9, 0x30, 0x29, 0xe5, 0xd7, 0xfb, 0xc8, 0x2f, 0x06,
	0x7a, 0x5e, 0x9c, 0x04, 0x55, 0x60, 0xe6, 0xe6, 0x9e, 0xba, 0xed, 0xa3, 0xad, 0x73, 0xdc, 0x66,
	0xa5, 0x2f, 0xb5, 0xeb, 0x7f, 0x77, 0xd1, 0xfa, 0xf1, 0x02, 0x48, 0x9c, 0xa2, 0xab, 0xf5, 0x04,
	0xf8, 0xae, 0xbf, 0xc8, 0xd3, 0x3f, 0x33, 0x6c, 0xeb, 0xde, 0x45, 0xe1, 0xaa, 0xba, 0xb7, 0xfd,
	0xf9, 0xf7, 0xdf, 0x6f, 0xee, 0x6d, 0xbc, 0x45, 0xa7, 0x7b, 0x4b, 0xb7, 0x45, 0x75, 0x5d, 0xe1,
	0x87, 0x83, 0x36, 0x57, 0xda, 0xc7, 0xbb, 0xcb, 0xc6, 0x17, 0xd1, 0x6a, 0x3d, 0xb8, 0x54, 0x37,
	0xeb, 0xc4, 0x2f, 0x3b, 0x69, 0x7b, 0x3b, 0x2b, 0x9d, 0xa8, 0x5a, 0x3b, 0x84, 0x59, 0xf6, 0xd0,
	0xe9, 0xb4, 0x9d, 0x9e, 0xf3, 0xe4, 0x4b, 0xe3, 0xeb, 0xe1, 0x1f, 0x17, 0xff, 0x72, 0x96, 0x01,
	0x79, 0x47, 0x08, 0xbd, 0x54, 0x20, 0x48, 0x89, 0x11, 0xdf, 0x4a, 0xac, 0x55, 0x66, 0x48, 0xa9,
	0x54, 0x20, 0xba, 0x25, 0x53, 0x3f, 0x82, 0x69, 0x6b, 0x67, 0xfe, 0xdf, 0x8d, 0xb8, 0x09, 0x73,
	0x63, 0x0e, 0xaa, 0xa7, 0x17, 0x6b, 0x99, 0x2b, 0xe3, 0x87, 0x32, 0xeb, 0xbc, 0x46, 0xf8, 0x50,
	0xb1, 0x30, 0x01, 0xd2, 0xf7, 0x7b, 0x64, 0xcc, 0x43, 0x28, 0x2e, 0xed, 0xa0, 0xb6, 0x8c, 0xb9,
	0x4d, 0xf2, 0x49, 0xa1, 0xa4, 0x55, 0xea, 0x5b, 0xa9, 0x63, 0x96, 0x81, 0x59, 0x28, 0x46, 0x27,
	0xa9, 0x9c, 0xd0, 0x8c, 0x15, 0x34, 0xe9, 0xf8, 0xe8, 0xe9, 0xe8, 0xc5, 0xf1, 0xa8, 0xdf, 0xd8,
	0xf3, 0x7b, 0x1d, 0xd7, 0x71, 0xfb, 0x1b, 0x4c, 0xa9, 0x94, 0x87, 0xe5, 0xab, 0xa5, 0xef, 0x8c,
	0x14, 0xc3, 0x95, 0x93, 0xe0, 0x31, 0x6a, 0x0c, 0x7a, 0x03, 0x3c, 0x40, 0x9d, 0x00, 0x6c, 0xae,
	0x05, 0x44, 0xe4, 0x24, 0x01, 0x41, 0x6c, 0x02, 0x44, 0x83, 0x91, 0xb9, 0x0e, 0x81, 0x44, 0x12,
	0x0c, 0x11, 0xd2, 0x12, 0xf8, 0xc0, 0x8d, 0xf5, 0xf1, 0x1a, 0xfa, 0xef, 0xa7, 0xeb, 0x5c, 0xd1,
	0xfb, 0xa8, 0x39, 0x87, 0x41, 0x9e, 0xc9, 0x30, 0xcf, 0x40, 0x54, 0x5b, 0x82, 0xb7, 0xcf, 0x47,
	0x43, 0x0d, 0xb7, 0x40, 0x23, 0x19, 0x1a, 0xfa, 0x66, 0xf7, 0x4c, 0x68, 0x61, 0xae, 0xd3, 0x55,
	0xe7, 0x6a, 0x32, 0x59, 0x2b, 0x17, 0xee, 0xd1, 0xbf, 0x00, 0x00, 0x00, 0xff, 0xff, 0x64, 0x1c,
	0xba, 0x01, 0x01, 0x04, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// SynchronizerClient is the client API for Synchronizer service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type SynchronizerClient interface {
	// Register associates this request with the current synchronization cycle and
	// returns an identifier for this registration. The caller returns this
	// identifier back in the evaluation request. This enables synchronizer to
	// identify stale evaluation requests belonging to a prior window.
	Register(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*RegisterResponse, error)
	// EvaluateProposals accepts a list of proposals and a registration identifier
	// for this request. If the synchronization cycle to which the request was
	// registered is completed, this request fails otherwise the proposals are
	// added to the list of proposals to be evaluated in the current cycle. At the
	//  end of the cycle, the user defined evaluation method is triggered and the
	// matches accepted by it are returned as results.
	EvaluateProposals(ctx context.Context, opts ...grpc.CallOption) (Synchronizer_EvaluateProposalsClient, error)
}

type synchronizerClient struct {
	cc *grpc.ClientConn
}

func NewSynchronizerClient(cc *grpc.ClientConn) SynchronizerClient {
	return &synchronizerClient{cc}
}

func (c *synchronizerClient) Register(ctx context.Context, in *RegisterRequest, opts ...grpc.CallOption) (*RegisterResponse, error) {
	out := new(RegisterResponse)
	err := c.cc.Invoke(ctx, "/api.internal.Synchronizer/Register", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *synchronizerClient) EvaluateProposals(ctx context.Context, opts ...grpc.CallOption) (Synchronizer_EvaluateProposalsClient, error) {
	stream, err := c.cc.NewStream(ctx, &_Synchronizer_serviceDesc.Streams[0], "/api.internal.Synchronizer/EvaluateProposals", opts...)
	if err != nil {
		return nil, err
	}
	x := &synchronizerEvaluateProposalsClient{stream}
	return x, nil
}

type Synchronizer_EvaluateProposalsClient interface {
	Send(*EvaluateProposalsRequest) error
	Recv() (*EvaluateProposalsResponse, error)
	grpc.ClientStream
}

type synchronizerEvaluateProposalsClient struct {
	grpc.ClientStream
}

func (x *synchronizerEvaluateProposalsClient) Send(m *EvaluateProposalsRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *synchronizerEvaluateProposalsClient) Recv() (*EvaluateProposalsResponse, error) {
	m := new(EvaluateProposalsResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// SynchronizerServer is the server API for Synchronizer service.
type SynchronizerServer interface {
	// Register associates this request with the current synchronization cycle and
	// returns an identifier for this registration. The caller returns this
	// identifier back in the evaluation request. This enables synchronizer to
	// identify stale evaluation requests belonging to a prior window.
	Register(context.Context, *RegisterRequest) (*RegisterResponse, error)
	// EvaluateProposals accepts a list of proposals and a registration identifier
	// for this request. If the synchronization cycle to which the request was
	// registered is completed, this request fails otherwise the proposals are
	// added to the list of proposals to be evaluated in the current cycle. At the
	//  end of the cycle, the user defined evaluation method is triggered and the
	// matches accepted by it are returned as results.
	EvaluateProposals(Synchronizer_EvaluateProposalsServer) error
}

// UnimplementedSynchronizerServer can be embedded to have forward compatible implementations.
type UnimplementedSynchronizerServer struct {
}

func (*UnimplementedSynchronizerServer) Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Register not implemented")
}
func (*UnimplementedSynchronizerServer) EvaluateProposals(srv Synchronizer_EvaluateProposalsServer) error {
	return status.Errorf(codes.Unimplemented, "method EvaluateProposals not implemented")
}

func RegisterSynchronizerServer(s *grpc.Server, srv SynchronizerServer) {
	s.RegisterService(&_Synchronizer_serviceDesc, srv)
}

func _Synchronizer_Register_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RegisterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SynchronizerServer).Register(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.internal.Synchronizer/Register",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SynchronizerServer).Register(ctx, req.(*RegisterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Synchronizer_EvaluateProposals_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(SynchronizerServer).EvaluateProposals(&synchronizerEvaluateProposalsServer{stream})
}

type Synchronizer_EvaluateProposalsServer interface {
	Send(*EvaluateProposalsResponse) error
	Recv() (*EvaluateProposalsRequest, error)
	grpc.ServerStream
}

type synchronizerEvaluateProposalsServer struct {
	grpc.ServerStream
}

func (x *synchronizerEvaluateProposalsServer) Send(m *EvaluateProposalsResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *synchronizerEvaluateProposalsServer) Recv() (*EvaluateProposalsRequest, error) {
	m := new(EvaluateProposalsRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

var _Synchronizer_serviceDesc = grpc.ServiceDesc{
	ServiceName: "api.internal.Synchronizer",
	HandlerType: (*SynchronizerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Register",
			Handler:    _Synchronizer_Register_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "EvaluateProposals",
			Handler:       _Synchronizer_EvaluateProposals_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "internal/api/synchronizer.proto",
}
