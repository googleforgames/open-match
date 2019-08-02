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
	Matches []*pb.Match `protobuf:"bytes,1,rep,name=matches,proto3" json:"matches,omitempty"`
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

func (m *EvaluateProposalsRequest) GetMatches() []*pb.Match {
	if m != nil {
		return m.Matches
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
	Matches              []*pb.Match `protobuf:"bytes,1,rep,name=matches,proto3" json:"matches,omitempty"`
	XXX_NoUnkeyedLiteral struct{}    `json:"-"`
	XXX_unrecognized     []byte      `json:"-"`
	XXX_sizecache        int32       `json:"-"`
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

func (m *EvaluateProposalsResponse) GetMatches() []*pb.Match {
	if m != nil {
		return m.Matches
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
	// 547 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x53, 0x4d, 0x6f, 0xd3, 0x40,
	0x10, 0x95, 0x1d, 0xd4, 0xc2, 0x52, 0x41, 0xbb, 0x07, 0x94, 0x86, 0xaf, 0xad, 0x8b, 0x4a, 0x15,
	0x11, 0x6f, 0x1b, 0x7a, 0x0a, 0x42, 0x6a, 0x81, 0x1e, 0x2a, 0x15, 0xa8, 0x5c, 0x89, 0x03, 0xb7,
	0x8d, 0x3d, 0xd8, 0x8b, 0xec, 0xdd, 0x65, 0x67, 0xdd, 0x02, 0x47, 0x6e, 0x5c, 0x29, 0x27, 0x7e,
	0x08, 0x7f, 0x84, 0x33, 0x37, 0x7e, 0x08, 0xb2, 0x13, 0xb7, 0x49, 0xd3, 0x0a, 0x4e, 0x96, 0x67,
	0xde, 0xbc, 0x99, 0xf7, 0x76, 0x86, 0xdc, 0x97, 0xca, 0x81, 0x55, 0x22, 0xe7, 0xc2, 0x48, 0x8e,
	0x9f, 0x54, 0x9c, 0x59, 0xad, 0xe4, 0x67, 0xb0, 0xa1, 0xb1, 0xda, 0x69, 0xba, 0x20, 0x8c, 0x0c,
	0x1b, 0x50, 0x87, 0x56, 0xa8, 0x02, 0x10, 0x45, 0x0a, 0x38, 0x42, 0x74, 0xee, 0xa4, 0x5a, 0xa7,
	0x39, 0xd4, 0x04, 0x42, 0x29, 0xed, 0x84, 0x93, 0x5a, 0x35, 0xd9, 0x47, 0xf5, 0x27, 0xee, 0xa5,
	0xa0, 0x7a, 0x78, 0x2c, 0xd2, 0x14, 0x2c, 0xd7, 0xa6, 0x46, 0xcc, 0xa2, 0x83, 0x25, 0x72, 0x33,
	0x82, 0x54, 0xa2, 0x03, 0x1b, 0xc1, 0x87, 0x12, 0xd0, 0x05, 0x01, 0x59, 0x3c, 0x0b, 0xa1, 0xd1,
	0x0a, 0x81, 0xde, 0x20, 0xbe, 0x4c, 0xda, 0x1e, 0xf3, 0xd6, 0xaf, 0x45, 0xbe, 0x4c, 0x82, 0x03,
	0xd2, 0xde, 0x3d, 0x12, 0x79, 0x29, 0x1c, 0x1c, 0x58, 0x6d, 0x34, 0x8a, 0x1c, 0xc7, 0xf5, 0xf4,
	0x01, 0x99, 0x2f, 0x84, 0x8b, 0x33, 0xc0, 0xb6, 0xc7, 0x5a, 0xeb, 0xd7, 0xfb, 0x24, 0xac, 0x24,
	0xbd, 0xac, 0x62, 0x51, 0x93, 0x1a, 0x33, 0xfa, 0xa7, 0x8c, 0x3b, 0x64, 0xf9, 0x02, 0xc6, 0x71,
	0xfb, 0xff, 0xa2, 0xec, 0x9f, 0xf8, 0x64, 0xe1, 0x70, 0xc2, 0x50, 0x9a, 0x93, 0xab, 0x8d, 0x12,
	0x7a, 0x37, 0x9c, 0xf4, 0x35, 0x3c, 0x27, 0xba, 0x73, 0xef, 0xb2, 0xf4, 0x68, 0x82, 0x60, 0xe5,
	0xcb, 0xaf, 0x3f, 0x27, 0xfe, 0x6d, 0xba, 0xcc, 0x8f, 0x36, 0xa7, 0x5e, 0x8d, 0xdb, 0xa6, 0xc3,
	0x77, 0x8f, 0x2c, 0xcd, 0x48, 0xa0, 0x6b, 0xd3, 0xc4, 0x97, 0xb9, 0xd6, 0x79, 0xf8, 0x4f, 0xdc,
	0x78, 0x92, 0xb0, 0x9e, 0x64, 0x3d, 0x58, 0x9d, 0x99, 0xc4, 0x34, 0xd8, 0x01, 0x8c, 0xab, 0x07,
	0x5e, 0xf7, 0xd9, 0xd7, 0xd6, 0xb7, 0x9d, 0xdf, 0x3e, 0xfd, 0xe9, 0x4d, 0x9b, 0x13, 0xec, 0x11,
	0xf2, 0xda, 0x80, 0x62, 0xb5, 0x89, 0xf4, 0x56, 0xe6, 0x9c, 0xc1, 0x01, 0xe7, 0xda, 0x80, 0xea,
	0xd5, 0x8e, 0x86, 0x09, 0x1c, 0x75, 0x56, 0xcf, 0xfe, 0x7b, 0x89, 0xc4, 0xb8, 0x44, 0xdc, 0x1e,
	0xad, 0x5f, 0x6a, 0x75, 0x69, 0x30, 0x8c, 0x75, 0xd1, 0x7d, 0x43, 0xe8, 0x8e, 0x11, 0x71, 0x06,
	0xac, 0x1f, 0x6e, 0xb0, 0x7d, 0x19, 0x43, 0xf5, 0x68, 0xdb, 0x0d, 0x65, 0x2a, 0x5d, 0x56, 0x0e,
	0x2b, 0x24, 0x1f, 0x95, 0xbe, 0xd3, 0x36, 0x15, 0x05, 0xe0, 0x44, 0x33, 0x3e, 0xcc, 0xf5, 0x90,
	0x17, 0xa2, 0x72, 0x92, 0xef, 0xef, 0x3d, 0xdf, 0x7d, 0x75, 0xb8, 0xdb, 0x6f, 0x6d, 0x86, 0x1b,
	0x5d, 0xdf, 0xf3, 0xfb, 0x8b, 0xc2, 0x98, 0x5c, 0xc6, 0xf5, 0xe6, 0xf2, 0xf7, 0xa8, 0xd5, 0x60,
	0x26, 0x12, 0x3d, 0x21, 0xad, 0xad, 0x8d, 0x2d, 0xba, 0x45, 0xba, 0x11, 0xb8, 0xd2, 0x2a, 0x48,
	0xd8, 0x71, 0x06, 0x8a, 0xb9, 0x0c, 0x98, 0x05, 0xd4, 0xa5, 0x8d, 0x81, 0x25, 0x1a, 0x90, 0x29,
	0xed, 0x18, 0x7c, 0x94, 0xe8, 0x42, 0x3a, 0x47, 0xae, 0xfc, 0xf0, 0xbd, 0x79, 0xfb, 0x94, 0xb4,
	0xcf, 0xcc, 0x60, 0x2f, 0x74, 0x5c, 0x16, 0xa0, 0x46, 0x97, 0x42, 0x57, 0x2e, 0xb6, 0x86, 0xa3,
	0x74, 0xc0, 0x13, 0x1d, 0x23, 0x7f, 0xbb, 0x76, 0x2e, 0x35, 0xa1, 0xeb, 0xf4, 0xdc, 0xa5, 0x19,
	0x0e, 0xe7, 0xea, 0xa3, 0x7b, 0xfc, 0x37, 0x00, 0x00, 0xff, 0xff, 0x7d, 0x34, 0xea, 0x8f, 0x05,
	0x04, 0x00, 0x00,
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
	EvaluateProposals(ctx context.Context, in *EvaluateProposalsRequest, opts ...grpc.CallOption) (*EvaluateProposalsResponse, error)
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

func (c *synchronizerClient) EvaluateProposals(ctx context.Context, in *EvaluateProposalsRequest, opts ...grpc.CallOption) (*EvaluateProposalsResponse, error) {
	out := new(EvaluateProposalsResponse)
	err := c.cc.Invoke(ctx, "/api.internal.Synchronizer/EvaluateProposals", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
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
	EvaluateProposals(context.Context, *EvaluateProposalsRequest) (*EvaluateProposalsResponse, error)
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

func _Synchronizer_EvaluateProposals_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EvaluateProposalsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SynchronizerServer).EvaluateProposals(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.internal.Synchronizer/EvaluateProposals",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SynchronizerServer).EvaluateProposals(ctx, req.(*EvaluateProposalsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Synchronizer_serviceDesc = grpc.ServiceDesc{
	ServiceName: "api.internal.Synchronizer",
	HandlerType: (*SynchronizerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Register",
			Handler:    _Synchronizer_Register_Handler,
		},
		{
			MethodName: "EvaluateProposals",
			Handler:    _Synchronizer_EvaluateProposals_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "internal/api/synchronizer.proto",
}
