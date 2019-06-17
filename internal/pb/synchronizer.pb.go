// Code generated by protoc-gen-go. DO NOT EDIT.
// source: api/synchronizer.proto

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

type GetContextRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetContextRequest) Reset()         { *m = GetContextRequest{} }
func (m *GetContextRequest) String() string { return proto.CompactTextString(m) }
func (*GetContextRequest) ProtoMessage()    {}
func (*GetContextRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_9dbd55595bca25da, []int{0}
}

func (m *GetContextRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetContextRequest.Unmarshal(m, b)
}
func (m *GetContextRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetContextRequest.Marshal(b, m, deterministic)
}
func (m *GetContextRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetContextRequest.Merge(m, src)
}
func (m *GetContextRequest) XXX_Size() int {
	return xxx_messageInfo_GetContextRequest.Size(m)
}
func (m *GetContextRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_GetContextRequest.DiscardUnknown(m)
}

var xxx_messageInfo_GetContextRequest proto.InternalMessageInfo

type GetContextResponse struct {
	// Context identifier for the current synchronization window.
	ContextId            string   `protobuf:"bytes,1,opt,name=context_id,json=contextId,proto3" json:"context_id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetContextResponse) Reset()         { *m = GetContextResponse{} }
func (m *GetContextResponse) String() string { return proto.CompactTextString(m) }
func (*GetContextResponse) ProtoMessage()    {}
func (*GetContextResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_9dbd55595bca25da, []int{1}
}

func (m *GetContextResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetContextResponse.Unmarshal(m, b)
}
func (m *GetContextResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetContextResponse.Marshal(b, m, deterministic)
}
func (m *GetContextResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetContextResponse.Merge(m, src)
}
func (m *GetContextResponse) XXX_Size() int {
	return xxx_messageInfo_GetContextResponse.Size(m)
}
func (m *GetContextResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_GetContextResponse.DiscardUnknown(m)
}

var xxx_messageInfo_GetContextResponse proto.InternalMessageInfo

func (m *GetContextResponse) GetContextId() string {
	if m != nil {
		return m.ContextId
	}
	return ""
}

type EvaluateRequest struct {
	// List of Matches to evaluate.
	Match                []*Match `protobuf:"bytes,1,rep,name=match,proto3" json:"match,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *EvaluateRequest) Reset()         { *m = EvaluateRequest{} }
func (m *EvaluateRequest) String() string { return proto.CompactTextString(m) }
func (*EvaluateRequest) ProtoMessage()    {}
func (*EvaluateRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_9dbd55595bca25da, []int{2}
}

func (m *EvaluateRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_EvaluateRequest.Unmarshal(m, b)
}
func (m *EvaluateRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_EvaluateRequest.Marshal(b, m, deterministic)
}
func (m *EvaluateRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EvaluateRequest.Merge(m, src)
}
func (m *EvaluateRequest) XXX_Size() int {
	return xxx_messageInfo_EvaluateRequest.Size(m)
}
func (m *EvaluateRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_EvaluateRequest.DiscardUnknown(m)
}

var xxx_messageInfo_EvaluateRequest proto.InternalMessageInfo

func (m *EvaluateRequest) GetMatch() []*Match {
	if m != nil {
		return m.Match
	}
	return nil
}

type EvaluateResponse struct {
	// Accepted list of Matches.
	Match                []*Match `protobuf:"bytes,1,rep,name=match,proto3" json:"match,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *EvaluateResponse) Reset()         { *m = EvaluateResponse{} }
func (m *EvaluateResponse) String() string { return proto.CompactTextString(m) }
func (*EvaluateResponse) ProtoMessage()    {}
func (*EvaluateResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_9dbd55595bca25da, []int{3}
}

func (m *EvaluateResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_EvaluateResponse.Unmarshal(m, b)
}
func (m *EvaluateResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_EvaluateResponse.Marshal(b, m, deterministic)
}
func (m *EvaluateResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EvaluateResponse.Merge(m, src)
}
func (m *EvaluateResponse) XXX_Size() int {
	return xxx_messageInfo_EvaluateResponse.Size(m)
}
func (m *EvaluateResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_EvaluateResponse.DiscardUnknown(m)
}

var xxx_messageInfo_EvaluateResponse proto.InternalMessageInfo

func (m *EvaluateResponse) GetMatch() []*Match {
	if m != nil {
		return m.Match
	}
	return nil
}

func init() {
	proto.RegisterType((*GetContextRequest)(nil), "api.GetContextRequest")
	proto.RegisterType((*GetContextResponse)(nil), "api.GetContextResponse")
	proto.RegisterType((*EvaluateRequest)(nil), "api.EvaluateRequest")
	proto.RegisterType((*EvaluateResponse)(nil), "api.EvaluateResponse")
}

func init() { proto.RegisterFile("api/synchronizer.proto", fileDescriptor_9dbd55595bca25da) }

var fileDescriptor_9dbd55595bca25da = []byte{
	// 527 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x84, 0x53, 0x41, 0x8b, 0xd3, 0x40,
	0x18, 0x25, 0xa9, 0xae, 0xee, 0xac, 0xe0, 0x3a, 0x6a, 0x2d, 0x41, 0x61, 0x36, 0x82, 0x2c, 0x65,
	0x9b, 0xe9, 0xb6, 0x3d, 0x55, 0x84, 0x5d, 0xd7, 0x22, 0x85, 0x55, 0xa1, 0x0b, 0x1e, 0xbc, 0x2c,
	0xd3, 0xc9, 0x67, 0x32, 0xd2, 0xcc, 0x8c, 0x33, 0x93, 0xee, 0xea, 0xd1, 0x9f, 0xa0, 0x37, 0x7f,
	0x88, 0x7f, 0x44, 0xf0, 0xe6, 0xcd, 0xb3, 0xbf, 0x41, 0x9a, 0xa4, 0xb4, 0x36, 0x82, 0xa7, 0x90,
	0xf7, 0xe6, 0xbd, 0x6f, 0xde, 0xcb, 0x17, 0xd4, 0x64, 0x5a, 0x50, 0xfb, 0x41, 0xf2, 0xd4, 0x28,
	0x29, 0x3e, 0x82, 0x89, 0xb4, 0x51, 0x4e, 0xe1, 0x06, 0xd3, 0x22, 0xc0, 0x0b, 0x32, 0x03, 0x6b,
	0x59, 0x02, 0xb6, 0x24, 0x82, 0xfb, 0x89, 0x52, 0xc9, 0x0c, 0xe8, 0x82, 0x62, 0x52, 0x2a, 0xc7,
	0x9c, 0x50, 0x72, 0xc9, 0x1e, 0x14, 0x0f, 0xde, 0x49, 0x40, 0x76, 0xec, 0x05, 0x4b, 0x12, 0x30,
	0x54, 0xe9, 0xe2, 0x44, 0xfd, 0x74, 0x78, 0x1b, 0xdd, 0x7a, 0x0e, 0xee, 0x44, 0x49, 0x07, 0x97,
	0x6e, 0x02, 0xef, 0x73, 0xb0, 0x2e, 0xec, 0x23, 0xbc, 0x0e, 0x5a, 0xad, 0xa4, 0x05, 0xfc, 0x00,
	0x21, 0x5e, 0x42, 0xe7, 0x22, 0x6e, 0x79, 0xc4, 0xdb, 0xdf, 0x9e, 0x6c, 0x57, 0xc8, 0x38, 0x0e,
	0xfb, 0xe8, 0xe6, 0x68, 0xce, 0x66, 0x39, 0x73, 0x50, 0xf9, 0x60, 0x82, 0xae, 0x66, 0xcc, 0xf1,
	0xb4, 0xe5, 0x91, 0xc6, 0xfe, 0x4e, 0x0f, 0x45, 0x4c, 0x8b, 0xe8, 0xc5, 0x02, 0x99, 0x94, 0x44,
	0x38, 0x40, 0xbb, 0x2b, 0x51, 0x35, 0xe7, 0xbf, 0xaa, 0xde, 0x0f, 0x0f, 0xdd, 0x38, 0x5b, 0x2b,
	0x0c, 0x9f, 0x23, 0xb4, 0xba, 0x30, 0x6e, 0x16, 0x8a, 0x5a, 0xac, 0xe0, 0x5e, 0x0d, 0x2f, 0x27,
	0x86, 0xe4, 0xd3, 0xf7, 0x5f, 0x5f, 0xfc, 0x00, 0xb7, 0xe8, 0xfc, 0xf0, 0xaf, 0x2f, 0x41, 0xab,
	0x78, 0x18, 0xd0, 0xf5, 0xe5, 0x3d, 0xf1, 0x9d, 0xc2, 0x66, 0x23, 0x6b, 0x70, 0x77, 0x03, 0xad,
	0xac, 0x0f, 0x0a, 0xeb, 0x47, 0xe1, 0x5e, 0xcd, 0xba, 0x88, 0x02, 0x76, 0x08, 0x95, 0x64, 0xe8,
	0xb5, 0x9f, 0xfe, 0xf6, 0x3f, 0x1f, 0xff, 0xf4, 0xf1, 0xb7, 0x8d, 0x7c, 0xe1, 0x18, 0xa1, 0x57,
	0x1a, 0x24, 0x29, 0x5a, 0xc0, 0xcd, 0xd4, 0x39, 0x6d, 0x87, 0x94, 0x2a, 0x0d, 0xb2, 0x53, 0xf8,
	0x44, 0x31, 0xcc, 0x83, 0x87, 0xab, 0xf7, 0x4e, 0x2c, 0x2c, 0xcf, 0xad, 0x3d, 0x2a, 0x57, 0x25,
	0x31, 0x2a, 0xd7, 0x36, 0xe2, 0x2a, 0x6b, 0xbf, 0x46, 0xf8, 0x58, 0x33, 0x9e, 0x02, 0xe9, 0x45,
	0x5d, 0x72, 0x2a, 0x38, 0x2c, 0x3a, 0x3f, 0x5a, 0x5a, 0x26, 0xc2, 0xa5, 0xf9, 0x74, 0x71, 0x92,
	0x96, 0xd2, 0xb7, 0xca, 0x24, 0x2c, 0x03, 0xbb, 0x36, 0x8c, 0x4e, 0x67, 0x6a, 0x4a, 0x33, 0x66,
	0x1d, 0x18, 0x7a, 0x3a, 0x3e, 0x19, 0xbd, 0x3c, 0x1b, 0xf5, 0x1a, 0x87, 0x51, 0xb7, 0xed, 0x7b,
	0x7e, 0x6f, 0x97, 0x69, 0x3d, 0x13, 0xbc, 0xd8, 0x32, 0xfa, 0xce, 0x2a, 0x39, 0xac, 0x21, 0x93,
	0xc7, 0xa8, 0x31, 0xe8, 0x0e, 0xf0, 0x00, 0xb5, 0x27, 0xe0, 0x72, 0x23, 0x21, 0x26, 0x17, 0x29,
	0x48, 0xe2, 0x52, 0x20, 0x06, 0xac, 0xca, 0x0d, 0x07, 0x12, 0x2b, 0xb0, 0x44, 0x2a, 0x47, 0xe0,
	0x52, 0x58, 0x17, 0xe1, 0x2d, 0x74, 0xe5, 0xab, 0xef, 0x5d, 0x33, 0x4f, 0x50, 0x6b, 0x55, 0x06,
	0x79, 0xa6, 0x78, 0x9e, 0x81, 0x2c, 0xb7, 0x1a, 0xef, 0xfd, 0xbb, 0x1a, 0x6a, 0x85, 0x03, 0x1a,
	0x2b, 0x6e, 0xe9, 0x9b, 0x1d, 0x21, 0x1d, 0x18, 0xc9, 0x66, 0x54, 0x4f, 0xa7, 0x5b, 0xc5, 0x5f,
	0xd0, 0xff, 0x13, 0x00, 0x00, 0xff, 0xff, 0x38, 0x5f, 0x08, 0x01, 0x84, 0x03, 0x00, 0x00,
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
	// GetContext returns the context for the synchronization window. The caller
	// requests for a context and then sends the context back in the evaluation
	// request. This enables identify stale evaluation requests belonging to a
	// prior window when synchronizing evaluation requests for a window.
	GetContext(ctx context.Context, in *GetContextRequest, opts ...grpc.CallOption) (*GetContextResponse, error)
	// Evaluate accepts a list of matches, triggers the user configured evaluation
	// function with these and other matches in the evaluation window and returns
	// matches that are accepted by the Evaluator as valid results.
	Evaluate(ctx context.Context, in *EvaluateRequest, opts ...grpc.CallOption) (*EvaluateResponse, error)
}

type synchronizerClient struct {
	cc *grpc.ClientConn
}

func NewSynchronizerClient(cc *grpc.ClientConn) SynchronizerClient {
	return &synchronizerClient{cc}
}

func (c *synchronizerClient) GetContext(ctx context.Context, in *GetContextRequest, opts ...grpc.CallOption) (*GetContextResponse, error) {
	out := new(GetContextResponse)
	err := c.cc.Invoke(ctx, "/api.Synchronizer/GetContext", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *synchronizerClient) Evaluate(ctx context.Context, in *EvaluateRequest, opts ...grpc.CallOption) (*EvaluateResponse, error) {
	out := new(EvaluateResponse)
	err := c.cc.Invoke(ctx, "/api.Synchronizer/Evaluate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SynchronizerServer is the server API for Synchronizer service.
type SynchronizerServer interface {
	// GetContext returns the context for the synchronization window. The caller
	// requests for a context and then sends the context back in the evaluation
	// request. This enables identify stale evaluation requests belonging to a
	// prior window when synchronizing evaluation requests for a window.
	GetContext(context.Context, *GetContextRequest) (*GetContextResponse, error)
	// Evaluate accepts a list of matches, triggers the user configured evaluation
	// function with these and other matches in the evaluation window and returns
	// matches that are accepted by the Evaluator as valid results.
	Evaluate(context.Context, *EvaluateRequest) (*EvaluateResponse, error)
}

func RegisterSynchronizerServer(s *grpc.Server, srv SynchronizerServer) {
	s.RegisterService(&_Synchronizer_serviceDesc, srv)
}

func _Synchronizer_GetContext_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetContextRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SynchronizerServer).GetContext(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.Synchronizer/GetContext",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SynchronizerServer).GetContext(ctx, req.(*GetContextRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Synchronizer_Evaluate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EvaluateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(SynchronizerServer).Evaluate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.Synchronizer/Evaluate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(SynchronizerServer).Evaluate(ctx, req.(*EvaluateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Synchronizer_serviceDesc = grpc.ServiceDesc{
	ServiceName: "api.Synchronizer",
	HandlerType: (*SynchronizerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetContext",
			Handler:    _Synchronizer_GetContext_Handler,
		},
		{
			MethodName: "Evaluate",
			Handler:    _Synchronizer_Evaluate_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api/synchronizer.proto",
}
