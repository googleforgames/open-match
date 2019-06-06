// Code generated by protoc-gen-go. DO NOT EDIT.
// source: api/backend.proto

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

// Configuration for a GRPC Match Function
type GrpcFunctionConfig struct {
	Host                 string   `protobuf:"bytes,1,opt,name=host,proto3" json:"host,omitempty"`
	Port                 int32    `protobuf:"varint,2,opt,name=port,proto3" json:"port,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GrpcFunctionConfig) Reset()         { *m = GrpcFunctionConfig{} }
func (m *GrpcFunctionConfig) String() string { return proto.CompactTextString(m) }
func (*GrpcFunctionConfig) ProtoMessage()    {}
func (*GrpcFunctionConfig) Descriptor() ([]byte, []int) {
	return fileDescriptor_8dab762378f455cd, []int{0}
}

func (m *GrpcFunctionConfig) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GrpcFunctionConfig.Unmarshal(m, b)
}
func (m *GrpcFunctionConfig) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GrpcFunctionConfig.Marshal(b, m, deterministic)
}
func (m *GrpcFunctionConfig) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GrpcFunctionConfig.Merge(m, src)
}
func (m *GrpcFunctionConfig) XXX_Size() int {
	return xxx_messageInfo_GrpcFunctionConfig.Size(m)
}
func (m *GrpcFunctionConfig) XXX_DiscardUnknown() {
	xxx_messageInfo_GrpcFunctionConfig.DiscardUnknown(m)
}

var xxx_messageInfo_GrpcFunctionConfig proto.InternalMessageInfo

func (m *GrpcFunctionConfig) GetHost() string {
	if m != nil {
		return m.Host
	}
	return ""
}

func (m *GrpcFunctionConfig) GetPort() int32 {
	if m != nil {
		return m.Port
	}
	return 0
}

// Configuration for a REST Match Function.
type RestFunctionConfig struct {
	Host                 string   `protobuf:"bytes,1,opt,name=host,proto3" json:"host,omitempty"`
	Port                 int32    `protobuf:"varint,2,opt,name=port,proto3" json:"port,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *RestFunctionConfig) Reset()         { *m = RestFunctionConfig{} }
func (m *RestFunctionConfig) String() string { return proto.CompactTextString(m) }
func (*RestFunctionConfig) ProtoMessage()    {}
func (*RestFunctionConfig) Descriptor() ([]byte, []int) {
	return fileDescriptor_8dab762378f455cd, []int{1}
}

func (m *RestFunctionConfig) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RestFunctionConfig.Unmarshal(m, b)
}
func (m *RestFunctionConfig) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RestFunctionConfig.Marshal(b, m, deterministic)
}
func (m *RestFunctionConfig) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RestFunctionConfig.Merge(m, src)
}
func (m *RestFunctionConfig) XXX_Size() int {
	return xxx_messageInfo_RestFunctionConfig.Size(m)
}
func (m *RestFunctionConfig) XXX_DiscardUnknown() {
	xxx_messageInfo_RestFunctionConfig.DiscardUnknown(m)
}

var xxx_messageInfo_RestFunctionConfig proto.InternalMessageInfo

func (m *RestFunctionConfig) GetHost() string {
	if m != nil {
		return m.Host
	}
	return ""
}

func (m *RestFunctionConfig) GetPort() int32 {
	if m != nil {
		return m.Port
	}
	return 0
}

// Configuration for the Match Function to be triggered by Open Match to
// generate proposals.
type FunctionConfig struct {
	// A developer-chosen human-readable name for this Match Function.
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// Properties for the type of this function.
	//
	// Types that are valid to be assigned to Type:
	//	*FunctionConfig_Grpc
	//	*FunctionConfig_Rest
	Type                 isFunctionConfig_Type `protobuf_oneof:"type"`
	XXX_NoUnkeyedLiteral struct{}              `json:"-"`
	XXX_unrecognized     []byte                `json:"-"`
	XXX_sizecache        int32                 `json:"-"`
}

func (m *FunctionConfig) Reset()         { *m = FunctionConfig{} }
func (m *FunctionConfig) String() string { return proto.CompactTextString(m) }
func (*FunctionConfig) ProtoMessage()    {}
func (*FunctionConfig) Descriptor() ([]byte, []int) {
	return fileDescriptor_8dab762378f455cd, []int{2}
}

func (m *FunctionConfig) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_FunctionConfig.Unmarshal(m, b)
}
func (m *FunctionConfig) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_FunctionConfig.Marshal(b, m, deterministic)
}
func (m *FunctionConfig) XXX_Merge(src proto.Message) {
	xxx_messageInfo_FunctionConfig.Merge(m, src)
}
func (m *FunctionConfig) XXX_Size() int {
	return xxx_messageInfo_FunctionConfig.Size(m)
}
func (m *FunctionConfig) XXX_DiscardUnknown() {
	xxx_messageInfo_FunctionConfig.DiscardUnknown(m)
}

var xxx_messageInfo_FunctionConfig proto.InternalMessageInfo

func (m *FunctionConfig) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

type isFunctionConfig_Type interface {
	isFunctionConfig_Type()
}

type FunctionConfig_Grpc struct {
	Grpc *GrpcFunctionConfig `protobuf:"bytes,10001,opt,name=grpc,proto3,oneof"`
}

type FunctionConfig_Rest struct {
	Rest *RestFunctionConfig `protobuf:"bytes,10002,opt,name=rest,proto3,oneof"`
}

func (*FunctionConfig_Grpc) isFunctionConfig_Type() {}

func (*FunctionConfig_Rest) isFunctionConfig_Type() {}

func (m *FunctionConfig) GetType() isFunctionConfig_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (m *FunctionConfig) GetGrpc() *GrpcFunctionConfig {
	if x, ok := m.GetType().(*FunctionConfig_Grpc); ok {
		return x.Grpc
	}
	return nil
}

func (m *FunctionConfig) GetRest() *RestFunctionConfig {
	if x, ok := m.GetType().(*FunctionConfig_Rest); ok {
		return x.Rest
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*FunctionConfig) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*FunctionConfig_Grpc)(nil),
		(*FunctionConfig_Rest)(nil),
	}
}

type FetchMatchesRequest struct {
	// Configuration of the MatchFunction to be executed for the given list of MatchProfiles
	Config *FunctionConfig `protobuf:"bytes,1,opt,name=config,proto3" json:"config,omitempty"`
	// MatchProfiles for which this MatchFunction should be executed.
	Profile              []*MatchProfile `protobuf:"bytes,2,rep,name=profile,proto3" json:"profile,omitempty"`
	XXX_NoUnkeyedLiteral struct{}        `json:"-"`
	XXX_unrecognized     []byte          `json:"-"`
	XXX_sizecache        int32           `json:"-"`
}

func (m *FetchMatchesRequest) Reset()         { *m = FetchMatchesRequest{} }
func (m *FetchMatchesRequest) String() string { return proto.CompactTextString(m) }
func (*FetchMatchesRequest) ProtoMessage()    {}
func (*FetchMatchesRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_8dab762378f455cd, []int{3}
}

func (m *FetchMatchesRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_FetchMatchesRequest.Unmarshal(m, b)
}
func (m *FetchMatchesRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_FetchMatchesRequest.Marshal(b, m, deterministic)
}
func (m *FetchMatchesRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_FetchMatchesRequest.Merge(m, src)
}
func (m *FetchMatchesRequest) XXX_Size() int {
	return xxx_messageInfo_FetchMatchesRequest.Size(m)
}
func (m *FetchMatchesRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_FetchMatchesRequest.DiscardUnknown(m)
}

var xxx_messageInfo_FetchMatchesRequest proto.InternalMessageInfo

func (m *FetchMatchesRequest) GetConfig() *FunctionConfig {
	if m != nil {
		return m.Config
	}
	return nil
}

func (m *FetchMatchesRequest) GetProfile() []*MatchProfile {
	if m != nil {
		return m.Profile
	}
	return nil
}

type FetchMatchesResponse struct {
	// Result Match for the requested MatchProfile.
	Match                *Match   `protobuf:"bytes,1,opt,name=match,proto3" json:"match,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *FetchMatchesResponse) Reset()         { *m = FetchMatchesResponse{} }
func (m *FetchMatchesResponse) String() string { return proto.CompactTextString(m) }
func (*FetchMatchesResponse) ProtoMessage()    {}
func (*FetchMatchesResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_8dab762378f455cd, []int{4}
}

func (m *FetchMatchesResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_FetchMatchesResponse.Unmarshal(m, b)
}
func (m *FetchMatchesResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_FetchMatchesResponse.Marshal(b, m, deterministic)
}
func (m *FetchMatchesResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_FetchMatchesResponse.Merge(m, src)
}
func (m *FetchMatchesResponse) XXX_Size() int {
	return xxx_messageInfo_FetchMatchesResponse.Size(m)
}
func (m *FetchMatchesResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_FetchMatchesResponse.DiscardUnknown(m)
}

var xxx_messageInfo_FetchMatchesResponse proto.InternalMessageInfo

func (m *FetchMatchesResponse) GetMatch() *Match {
	if m != nil {
		return m.Match
	}
	return nil
}

type AssignTicketsRequest struct {
	// List of Ticket IDs for which the Assignment is to be made.
	TicketId []string `protobuf:"bytes,1,rep,name=ticket_id,json=ticketId,proto3" json:"ticket_id,omitempty"`
	// Assignment to be associated with the Ticket IDs.
	Assignment           *Assignment `protobuf:"bytes,2,opt,name=assignment,proto3" json:"assignment,omitempty"`
	XXX_NoUnkeyedLiteral struct{}    `json:"-"`
	XXX_unrecognized     []byte      `json:"-"`
	XXX_sizecache        int32       `json:"-"`
}

func (m *AssignTicketsRequest) Reset()         { *m = AssignTicketsRequest{} }
func (m *AssignTicketsRequest) String() string { return proto.CompactTextString(m) }
func (*AssignTicketsRequest) ProtoMessage()    {}
func (*AssignTicketsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_8dab762378f455cd, []int{5}
}

func (m *AssignTicketsRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AssignTicketsRequest.Unmarshal(m, b)
}
func (m *AssignTicketsRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AssignTicketsRequest.Marshal(b, m, deterministic)
}
func (m *AssignTicketsRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AssignTicketsRequest.Merge(m, src)
}
func (m *AssignTicketsRequest) XXX_Size() int {
	return xxx_messageInfo_AssignTicketsRequest.Size(m)
}
func (m *AssignTicketsRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_AssignTicketsRequest.DiscardUnknown(m)
}

var xxx_messageInfo_AssignTicketsRequest proto.InternalMessageInfo

func (m *AssignTicketsRequest) GetTicketId() []string {
	if m != nil {
		return m.TicketId
	}
	return nil
}

func (m *AssignTicketsRequest) GetAssignment() *Assignment {
	if m != nil {
		return m.Assignment
	}
	return nil
}

type AssignTicketsResponse struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AssignTicketsResponse) Reset()         { *m = AssignTicketsResponse{} }
func (m *AssignTicketsResponse) String() string { return proto.CompactTextString(m) }
func (*AssignTicketsResponse) ProtoMessage()    {}
func (*AssignTicketsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_8dab762378f455cd, []int{6}
}

func (m *AssignTicketsResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AssignTicketsResponse.Unmarshal(m, b)
}
func (m *AssignTicketsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AssignTicketsResponse.Marshal(b, m, deterministic)
}
func (m *AssignTicketsResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AssignTicketsResponse.Merge(m, src)
}
func (m *AssignTicketsResponse) XXX_Size() int {
	return xxx_messageInfo_AssignTicketsResponse.Size(m)
}
func (m *AssignTicketsResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_AssignTicketsResponse.DiscardUnknown(m)
}

var xxx_messageInfo_AssignTicketsResponse proto.InternalMessageInfo

func init() {
	proto.RegisterType((*GrpcFunctionConfig)(nil), "api.GrpcFunctionConfig")
	proto.RegisterType((*RestFunctionConfig)(nil), "api.RestFunctionConfig")
	proto.RegisterType((*FunctionConfig)(nil), "api.FunctionConfig")
	proto.RegisterType((*FetchMatchesRequest)(nil), "api.FetchMatchesRequest")
	proto.RegisterType((*FetchMatchesResponse)(nil), "api.FetchMatchesResponse")
	proto.RegisterType((*AssignTicketsRequest)(nil), "api.AssignTicketsRequest")
	proto.RegisterType((*AssignTicketsResponse)(nil), "api.AssignTicketsResponse")
}

func init() { proto.RegisterFile("api/backend.proto", fileDescriptor_8dab762378f455cd) }

var fileDescriptor_8dab762378f455cd = []byte{
	// 696 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x54, 0xcd, 0x72, 0xf3, 0x34,
	0x14, 0xc5, 0x4e, 0xbe, 0x94, 0x2a, 0xfc, 0x7d, 0x6a, 0xa1, 0x6e, 0x60, 0x40, 0x18, 0x98, 0xc9,
	0xa4, 0xc4, 0x4a, 0x4d, 0x17, 0x4c, 0x80, 0x99, 0xa6, 0xa5, 0x85, 0xce, 0x94, 0xc2, 0x18, 0x86,
	0x05, 0x1b, 0x46, 0x91, 0x6f, 0x6c, 0xd1, 0x58, 0x12, 0x96, 0xdc, 0xc2, 0x96, 0x35, 0x1b, 0xca,
	0x8e, 0xb7, 0xe0, 0x59, 0xd8, 0xf0, 0x00, 0xb0, 0xe0, 0x2d, 0x18, 0xcb, 0x4d, 0x93, 0x90, 0xae,
	0x58, 0x45, 0x39, 0xf7, 0x9c, 0x73, 0x8f, 0xae, 0x24, 0xa3, 0xa7, 0x4c, 0x0b, 0x3a, 0x65, 0xfc,
	0x1a, 0x64, 0x1a, 0xe9, 0x52, 0x59, 0x85, 0x5b, 0x4c, 0x8b, 0x1e, 0xae, 0xf1, 0x02, 0x8c, 0x61,
	0x19, 0x98, 0xa6, 0xd0, 0x7b, 0x2d, 0x53, 0x2a, 0x9b, 0x03, 0xad, 0x4b, 0x4c, 0x4a, 0x65, 0x99,
	0x15, 0x4a, 0x2e, 0xaa, 0xef, 0xba, 0x1f, 0x3e, 0xcc, 0x40, 0x0e, 0xcd, 0x2d, 0xcb, 0x32, 0x28,
	0xa9, 0xd2, 0x8e, 0xb1, 0xc9, 0x0e, 0x3f, 0x44, 0xf8, 0x93, 0x52, 0xf3, 0xf3, 0x4a, 0xf2, 0x1a,
	0x3e, 0x55, 0x72, 0x26, 0x32, 0x8c, 0x51, 0x3b, 0x57, 0xc6, 0x06, 0x1e, 0xf1, 0xfa, 0xdb, 0x89,
	0x5b, 0xd7, 0x98, 0x56, 0xa5, 0x0d, 0x7c, 0xe2, 0xf5, 0x9f, 0x24, 0x6e, 0x5d, 0xab, 0x13, 0x30,
	0xf6, 0x7f, 0xaa, 0x7f, 0xf6, 0xd0, 0x0b, 0x9b, 0x52, 0xc9, 0x0a, 0x58, 0x48, 0xeb, 0x35, 0x8e,
	0x50, 0x3b, 0x2b, 0x35, 0x0f, 0x7e, 0xb9, 0x22, 0x5e, 0xbf, 0x1b, 0xef, 0x45, 0x4c, 0x8b, 0x68,
	0x33, 0xf4, 0xa7, 0xcf, 0x24, 0x8e, 0x57, 0xf3, 0x4b, 0x30, 0x36, 0xb8, 0x5b, 0xe5, 0x6f, 0xc6,
	0xac, 0xf9, 0x35, 0xef, 0xa4, 0x83, 0xda, 0xf6, 0x47, 0x0d, 0xa1, 0x42, 0x3b, 0xe7, 0x60, 0x79,
	0xfe, 0x19, 0xb3, 0x3c, 0x07, 0x93, 0xc0, 0xf7, 0x15, 0x18, 0x8b, 0x0f, 0x50, 0x87, 0x3b, 0x81,
	0x0b, 0xd5, 0x8d, 0x77, 0x9c, 0xdf, 0xba, 0x57, 0x72, 0x4f, 0xc1, 0x07, 0x68, 0x4b, 0x97, 0x6a,
	0x26, 0xe6, 0x10, 0xf8, 0xa4, 0xd5, 0xef, 0xc6, 0x4f, 0x1d, 0xdb, 0x59, 0x7e, 0xd1, 0x14, 0x92,
	0x05, 0x23, 0x7c, 0x1f, 0xed, 0xae, 0x37, 0x34, 0x5a, 0x49, 0x03, 0x98, 0xa0, 0x27, 0x45, 0x0d,
	0xdd, 0x37, 0x44, 0x4b, 0x8b, 0xa4, 0x29, 0x84, 0x29, 0xda, 0x9d, 0x18, 0x23, 0x32, 0xf9, 0x95,
	0xe0, 0xd7, 0x60, 0x1f, 0xb2, 0xbe, 0x8a, 0xb6, 0xad, 0x43, 0xbe, 0x15, 0x69, 0xe0, 0x91, 0x56,
	0x7f, 0x3b, 0x79, 0xb6, 0x01, 0x2e, 0x52, 0x4c, 0x11, 0x62, 0x4e, 0x54, 0x80, 0x6c, 0x0e, 0xa2,
	0x1b, 0xbf, 0xe8, 0xbc, 0x27, 0x0f, 0x70, 0xb2, 0x42, 0x09, 0xf7, 0xd0, 0xcb, 0xff, 0xe9, 0xd2,
	0x04, 0x8c, 0xff, 0xf6, 0xd0, 0xd6, 0x49, 0x73, 0x57, 0xf1, 0x35, 0x7a, 0x6e, 0x75, 0x13, 0x38,
	0x68, 0xc6, 0xb3, 0x39, 0xc8, 0xde, 0xfe, 0x23, 0x95, 0xc6, 0x30, 0x7c, 0xfb, 0xa7, 0x3f, 0xfe,
	0xfa, 0xd5, 0x7f, 0x3d, 0xdc, 0xa7, 0x37, 0x87, 0x8b, 0x57, 0x40, 0x8b, 0x86, 0x34, 0x9e, 0xd5,
	0x8a, 0xb1, 0x37, 0x18, 0x79, 0xb8, 0x40, 0xcf, 0xaf, 0x25, 0xc2, 0xfb, 0x2b, 0xf9, 0xd7, 0x67,
	0xd1, 0xeb, 0x3d, 0x56, 0xba, 0xef, 0xf7, 0x8e, 0xeb, 0xf7, 0x46, 0xd8, 0x5b, 0xed, 0xd7, 0x0c,
	0xca, 0x8c, 0x9b, 0x09, 0x8c, 0xbd, 0xc1, 0xc9, 0x3f, 0xfe, 0xdd, 0xe4, 0x4f, 0x1f, 0xff, 0xbe,
	0xdc, 0x6e, 0x78, 0x81, 0xd0, 0xe7, 0x1a, 0x24, 0x71, 0x3b, 0xc0, 0xaf, 0xe4, 0xd6, 0x6a, 0x33,
	0xa6, 0x54, 0x69, 0x90, 0x43, 0x17, 0x38, 0x4a, 0xe1, 0xa6, 0xf7, 0xd6, 0xf2, 0xff, 0x30, 0x15,
	0x86, 0x57, 0xc6, 0x1c, 0x37, 0x8f, 0x36, 0x2b, 0x55, 0xa5, 0x4d, 0xc4, 0x55, 0x31, 0xf8, 0x1a,
	0xe1, 0x89, 0x66, 0x3c, 0x07, 0x12, 0x47, 0x23, 0x72, 0x29, 0x38, 0xd4, 0x87, 0x7f, 0xbc, 0xb0,
	0xcc, 0x84, 0xcd, 0xab, 0x69, 0xcd, 0xa4, 0x8d, 0x74, 0xa6, 0xca, 0x8c, 0x15, 0x60, 0x56, 0x9a,
	0xd1, 0xe9, 0x5c, 0x4d, 0x69, 0xc1, 0x8c, 0x85, 0x92, 0x5e, 0x5e, 0x9c, 0x9e, 0x5d, 0x7d, 0x79,
	0x16, 0xb7, 0x0e, 0xa3, 0xd1, 0xc0, 0xf7, 0xfc, 0xf8, 0x25, 0xa6, 0xf5, 0x5c, 0x70, 0xf7, 0xde,
	0xe9, 0x77, 0x46, 0xc9, 0xf1, 0x06, 0x92, 0x7c, 0x80, 0x5a, 0x47, 0xa3, 0x23, 0x7c, 0x84, 0x06,
	0x09, 0xd8, 0xaa, 0x94, 0x90, 0x92, 0xdb, 0x1c, 0x24, 0xb1, 0x39, 0x90, 0x12, 0x8c, 0xaa, 0x4a,
	0x0e, 0x24, 0x55, 0x60, 0x88, 0x54, 0x96, 0xc0, 0x0f, 0xc2, 0xd8, 0x08, 0x77, 0x50, 0xfb, 0x37,
	0xdf, 0xdb, 0x2a, 0x3f, 0x42, 0xc1, 0x72, 0x18, 0xe4, 0x63, 0xc5, 0xab, 0xfa, 0xde, 0x38, 0x77,
	0xfc, 0xe6, 0xe3, 0xa3, 0xa1, 0x46, 0x58, 0xa0, 0xa9, 0xe2, 0x86, 0x7e, 0xd3, 0x15, 0xd2, 0x42,
	0x29, 0xd9, 0x9c, 0xea, 0xe9, 0xb4, 0xe3, 0xbe, 0x47, 0xef, 0xfd, 0x1b, 0x00, 0x00, 0xff, 0xff,
	0x7f, 0x8a, 0x6f, 0xd2, 0x09, 0x05, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// BackendClient is the client API for Backend service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type BackendClient interface {
	// FetchMatch triggers execution of the specfied MatchFunction for each of the
	// specified MatchProfiles. Each MatchFunction execution returns a set of
	// proposals which are then evaluated to generate results. FetchMatch method
	// streams these results back to the caller.
	FetchMatches(ctx context.Context, in *FetchMatchesRequest, opts ...grpc.CallOption) (Backend_FetchMatchesClient, error)
	// AssignTickets sets the specified Assignment on the Tickets for the Ticket
	// IDs passed.
	AssignTickets(ctx context.Context, in *AssignTicketsRequest, opts ...grpc.CallOption) (*AssignTicketsResponse, error)
}

type backendClient struct {
	cc *grpc.ClientConn
}

func NewBackendClient(cc *grpc.ClientConn) BackendClient {
	return &backendClient{cc}
}

func (c *backendClient) FetchMatches(ctx context.Context, in *FetchMatchesRequest, opts ...grpc.CallOption) (Backend_FetchMatchesClient, error) {
	stream, err := c.cc.NewStream(ctx, &_Backend_serviceDesc.Streams[0], "/api.Backend/FetchMatches", opts...)
	if err != nil {
		return nil, err
	}
	x := &backendFetchMatchesClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Backend_FetchMatchesClient interface {
	Recv() (*FetchMatchesResponse, error)
	grpc.ClientStream
}

type backendFetchMatchesClient struct {
	grpc.ClientStream
}

func (x *backendFetchMatchesClient) Recv() (*FetchMatchesResponse, error) {
	m := new(FetchMatchesResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *backendClient) AssignTickets(ctx context.Context, in *AssignTicketsRequest, opts ...grpc.CallOption) (*AssignTicketsResponse, error) {
	out := new(AssignTicketsResponse)
	err := c.cc.Invoke(ctx, "/api.Backend/AssignTickets", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BackendServer is the server API for Backend service.
type BackendServer interface {
	// FetchMatch triggers execution of the specfied MatchFunction for each of the
	// specified MatchProfiles. Each MatchFunction execution returns a set of
	// proposals which are then evaluated to generate results. FetchMatch method
	// streams these results back to the caller.
	FetchMatches(*FetchMatchesRequest, Backend_FetchMatchesServer) error
	// AssignTickets sets the specified Assignment on the Tickets for the Ticket
	// IDs passed.
	AssignTickets(context.Context, *AssignTicketsRequest) (*AssignTicketsResponse, error)
}

func RegisterBackendServer(s *grpc.Server, srv BackendServer) {
	s.RegisterService(&_Backend_serviceDesc, srv)
}

func _Backend_FetchMatches_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(FetchMatchesRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(BackendServer).FetchMatches(m, &backendFetchMatchesServer{stream})
}

type Backend_FetchMatchesServer interface {
	Send(*FetchMatchesResponse) error
	grpc.ServerStream
}

type backendFetchMatchesServer struct {
	grpc.ServerStream
}

func (x *backendFetchMatchesServer) Send(m *FetchMatchesResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _Backend_AssignTickets_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AssignTicketsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BackendServer).AssignTickets(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.Backend/AssignTickets",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BackendServer).AssignTickets(ctx, req.(*AssignTicketsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Backend_serviceDesc = grpc.ServiceDesc{
	ServiceName: "api.Backend",
	HandlerType: (*BackendServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AssignTickets",
			Handler:    _Backend_AssignTickets_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "FetchMatches",
			Handler:       _Backend_FetchMatches_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "api/backend.proto",
}
