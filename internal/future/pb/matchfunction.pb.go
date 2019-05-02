// Code generated by protoc-gen-go. DO NOT EDIT.
// source: api/matchfunction.proto

package pb

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger/options"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	grpc "google.golang.org/grpc"
	math "math"
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

type RunRequest struct {
	// The MatchProfile that describes the Match that this MatchFunction needs to
	// generate proposals for.
	Profile              *MatchProfile `protobuf:"bytes,1,opt,name=profile,proto3" json:"profile,omitempty"`
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
	XXX_unrecognized     []byte        `json:"-"`
	XXX_sizecache        int32         `json:"-"`
}

func (m *RunRequest) Reset()         { *m = RunRequest{} }
func (m *RunRequest) String() string { return proto.CompactTextString(m) }
func (*RunRequest) ProtoMessage()    {}
func (*RunRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_2b5069a21f149a55, []int{0}
}

func (m *RunRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RunRequest.Unmarshal(m, b)
}
func (m *RunRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RunRequest.Marshal(b, m, deterministic)
}
func (m *RunRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RunRequest.Merge(m, src)
}
func (m *RunRequest) XXX_Size() int {
	return xxx_messageInfo_RunRequest.Size(m)
}
func (m *RunRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_RunRequest.DiscardUnknown(m)
}

var xxx_messageInfo_RunRequest proto.InternalMessageInfo

func (m *RunRequest) GetProfile() *MatchProfile {
	if m != nil {
		return m.Profile
	}
	return nil
}

type RunResponse struct {
	// The proposal generated by this MatchFunction Run.
	Proposal             *Match   `protobuf:"bytes,1,opt,name=proposal,proto3" json:"proposal,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *RunResponse) Reset()         { *m = RunResponse{} }
func (m *RunResponse) String() string { return proto.CompactTextString(m) }
func (*RunResponse) ProtoMessage()    {}
func (*RunResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_2b5069a21f149a55, []int{1}
}

func (m *RunResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RunResponse.Unmarshal(m, b)
}
func (m *RunResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RunResponse.Marshal(b, m, deterministic)
}
func (m *RunResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RunResponse.Merge(m, src)
}
func (m *RunResponse) XXX_Size() int {
	return xxx_messageInfo_RunResponse.Size(m)
}
func (m *RunResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_RunResponse.DiscardUnknown(m)
}

var xxx_messageInfo_RunResponse proto.InternalMessageInfo

func (m *RunResponse) GetProposal() *Match {
	if m != nil {
		return m.Proposal
	}
	return nil
}

func init() {
	proto.RegisterType((*RunRequest)(nil), "api.RunRequest")
	proto.RegisterType((*RunResponse)(nil), "api.RunResponse")
}

func init() { proto.RegisterFile("api/matchfunction.proto", fileDescriptor_2b5069a21f149a55) }

var fileDescriptor_2b5069a21f149a55 = []byte{
	// 481 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x52, 0x4d, 0x6e, 0xd3, 0x40,
	0x14, 0x96, 0x1d, 0xd4, 0xa2, 0xa9, 0x80, 0x32, 0x12, 0x50, 0x45, 0x2c, 0x86, 0x20, 0x21, 0x14,
	0x88, 0x27, 0x0d, 0x65, 0x41, 0x10, 0x12, 0xa5, 0x04, 0x54, 0x29, 0x40, 0x65, 0x76, 0x20, 0x16,
	0x93, 0xf1, 0x8b, 0x3d, 0xc8, 0x99, 0x37, 0xcc, 0x4f, 0xcb, 0x9a, 0x23, 0xc0, 0x8e, 0xbb, 0x70,
	0x0a, 0xae, 0xd0, 0x63, 0xb0, 0x40, 0x1e, 0x37, 0x18, 0x54, 0x56, 0x96, 0xbf, 0x9f, 0xf7, 0x3d,
	0x7f, 0x7e, 0xe4, 0x86, 0x30, 0x8a, 0xaf, 0x84, 0x97, 0xd5, 0x32, 0x68, 0xe9, 0x15, 0xea, 0xcc,
	0x58, 0xf4, 0x48, 0x7b, 0xc2, 0xa8, 0x3e, 0x8d, 0x2c, 0x38, 0x27, 0x4a, 0x70, 0x2d, 0xd1, 0xbf,
	0x59, 0x22, 0x96, 0x35, 0xf0, 0x86, 0x12, 0x5a, 0xa3, 0x17, 0x8d, 0x6b, 0xcd, 0xde, 0x8f, 0x0f,
	0x39, 0x2a, 0x41, 0x8f, 0xdc, 0x89, 0x28, 0x4b, 0xb0, 0x1c, 0x4d, 0x54, 0x9c, 0x57, 0x0f, 0x1e,
	0x11, 0x92, 0x07, 0x9d, 0xc3, 0xa7, 0x00, 0xce, 0xd3, 0x7b, 0x64, 0xd3, 0x58, 0x5c, 0xaa, 0x1a,
	0x76, 0x12, 0x96, 0xdc, 0xdd, 0x9a, 0x5c, 0xcd, 0x84, 0x51, 0xd9, 0xab, 0x66, 0xbb, 0xa3, 0x96,
	0xc8, 0xd7, 0x8a, 0xc1, 0x43, 0xb2, 0x15, 0xad, 0xce, 0xa0, 0x76, 0x40, 0xef, 0x90, 0x8b, 0xc6,
	0xa2, 0x41, 0x27, 0xea, 0x33, 0x33, 0xe9, 0xcc, 0xf9, 0x1f, 0x6e, 0xf2, 0x81, 0x5c, 0x8a, 0xd0,
	0x8b, 0xb3, 0xaf, 0xa5, 0x73, 0xd2, 0xcb, 0x83, 0xa6, 0x57, 0xa2, 0xba, 0x5b, 0xa6, 0xbf, 0xdd,
	0x01, 0x6d, 0xc4, 0x80, 0x7d, 0xf9, 0x79, 0xfa, 0x2d, 0xed, 0x0f, 0xae, 0xf1, 0xe3, 0xdd, 0x7f,
	0x2b, 0xe3, 0x36, 0xe8, 0x69, 0x32, 0x1c, 0x27, 0xcf, 0x7e, 0xa5, 0x5f, 0xf7, 0x4f, 0x53, 0xfa,
	0x23, 0x21, 0x97, 0x63, 0x0c, 0x5b, 0xe7, 0x0c, 0x0e, 0x09, 0x79, 0x63, 0x40, 0xb3, 0x08, 0xd3,
	0xeb, 0x95, 0xf7, 0xc6, 0x4d, 0x39, 0x47, 0x03, 0x7a, 0x14, 0x87, 0x65, 0x05, 0x1c, 0xf7, 0x6f,
	0x77, 0xef, 0xa3, 0x42, 0x39, 0x19, 0x9c, 0x7b, 0xda, 0xf6, 0x5d, 0x5a, 0x0c, 0xc6, 0x65, 0x12,
	0x57, 0xc3, 0xf7, 0x84, 0xee, 0x1b, 0x21, 0x2b, 0x60, 0x93, 0x6c, 0xcc, 0xe6, 0x4a, 0x42, 0xd3,
	0xc0, 0x6c, 0x3d, 0xb2, 0x54, 0xbe, 0x0a, 0x8b, 0x46, 0xc9, 0x5f, 0x46, 0xeb, 0x41, 0x8d, 0xa1,
	0x38, 0xaa, 0x85, 0x5f, 0xa2, 0x5d, 0xfd, 0x95, 0xc8, 0x17, 0x35, 0x2e, 0xf8, 0x4a, 0x38, 0x0f,
	0x96, 0xcf, 0x0f, 0x0f, 0x66, 0xaf, 0xdf, 0xce, 0x26, 0xbd, 0xdd, 0x6c, 0x3c, 0x4c, 0x93, 0x74,
	0xb2, 0x2d, 0x8c, 0xa9, 0x95, 0x8c, 0xff, 0x8b, 0x7f, 0x74, 0xa8, 0xa7, 0xe7, 0x90, 0xfc, 0x31,
	0xe9, 0xed, 0x8d, 0xf7, 0xe8, 0x1e, 0x19, 0xe6, 0xe0, 0x83, 0xd5, 0x50, 0xb0, 0x93, 0x0a, 0x34,
	0xf3, 0x15, 0x30, 0x0b, 0x0e, 0x83, 0x95, 0xc0, 0x0a, 0x04, 0xc7, 0x34, 0x7a, 0x06, 0x9f, 0x95,
	0xf3, 0x19, 0xdd, 0x20, 0x17, 0xbe, 0xa7, 0xc9, 0xa6, 0x7d, 0x42, 0x76, 0xba, 0x46, 0xd8, 0x73,
	0x94, 0x61, 0x05, 0xba, 0xbd, 0x0f, 0x7a, 0xeb, 0xff, 0xfd, 0x70, 0xa7, 0x3c, 0xf0, 0x02, 0xa5,
	0xe3, 0xef, 0xa8, 0xd2, 0x1e, 0xac, 0x16, 0x35, 0x5f, 0x06, 0x1f, 0x2c, 0x70, 0xb3, 0x58, 0x6c,
	0xc4, 0xb3, 0x7a, 0xf0, 0x3b, 0x00, 0x00, 0xff, 0xff, 0x5a, 0xad, 0x8e, 0x08, 0xd6, 0x02, 0x00,
	0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// MatchFunctionClient is the client API for MatchFunction service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type MatchFunctionClient interface {
	// This is the function that is executed when by the Open Match backend to
	// generate Match proposals.
	Run(ctx context.Context, in *RunRequest, opts ...grpc.CallOption) (MatchFunction_RunClient, error)
}

type matchFunctionClient struct {
	cc *grpc.ClientConn
}

func NewMatchFunctionClient(cc *grpc.ClientConn) MatchFunctionClient {
	return &matchFunctionClient{cc}
}

func (c *matchFunctionClient) Run(ctx context.Context, in *RunRequest, opts ...grpc.CallOption) (MatchFunction_RunClient, error) {
	stream, err := c.cc.NewStream(ctx, &_MatchFunction_serviceDesc.Streams[0], "/api.MatchFunction/Run", opts...)
	if err != nil {
		return nil, err
	}
	x := &matchFunctionRunClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type MatchFunction_RunClient interface {
	Recv() (*RunResponse, error)
	grpc.ClientStream
}

type matchFunctionRunClient struct {
	grpc.ClientStream
}

func (x *matchFunctionRunClient) Recv() (*RunResponse, error) {
	m := new(RunResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// MatchFunctionServer is the server API for MatchFunction service.
type MatchFunctionServer interface {
	// This is the function that is executed when by the Open Match backend to
	// generate Match proposals.
	Run(*RunRequest, MatchFunction_RunServer) error
}

func RegisterMatchFunctionServer(s *grpc.Server, srv MatchFunctionServer) {
	s.RegisterService(&_MatchFunction_serviceDesc, srv)
}

func _MatchFunction_Run_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(RunRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(MatchFunctionServer).Run(m, &matchFunctionRunServer{stream})
}

type MatchFunction_RunServer interface {
	Send(*RunResponse) error
	grpc.ServerStream
}

type matchFunctionRunServer struct {
	grpc.ServerStream
}

func (x *matchFunctionRunServer) Send(m *RunResponse) error {
	return x.ServerStream.SendMsg(m)
}

var _MatchFunction_serviceDesc = grpc.ServiceDesc{
	ServiceName: "api.MatchFunction",
	HandlerType: (*MatchFunctionServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Run",
			Handler:       _MatchFunction_Run_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "api/matchfunction.proto",
}
