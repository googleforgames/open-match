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
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
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

type FunctionConfig_Type int32

const (
	FunctionConfig_GRPC FunctionConfig_Type = 0
	FunctionConfig_REST FunctionConfig_Type = 1
)

var FunctionConfig_Type_name = map[int32]string{
	0: "GRPC",
	1: "REST",
}

var FunctionConfig_Type_value = map[string]int32{
	"GRPC": 0,
	"REST": 1,
}

func (x FunctionConfig_Type) String() string {
	return proto.EnumName(FunctionConfig_Type_name, int32(x))
}

func (FunctionConfig_Type) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_8dab762378f455cd, []int{0, 0}
}

type AssignmentFailure_Cause int32

const (
	AssignmentFailure_UNKNOWN          AssignmentFailure_Cause = 0
	AssignmentFailure_TICKET_NOT_FOUND AssignmentFailure_Cause = 1
)

var AssignmentFailure_Cause_name = map[int32]string{
	0: "UNKNOWN",
	1: "TICKET_NOT_FOUND",
}

var AssignmentFailure_Cause_value = map[string]int32{
	"UNKNOWN":          0,
	"TICKET_NOT_FOUND": 1,
}

func (x AssignmentFailure_Cause) String() string {
	return proto.EnumName(AssignmentFailure_Cause_name, int32(x))
}

func (AssignmentFailure_Cause) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_8dab762378f455cd, []int{6, 0}
}

// FunctionConfig specifies a MMF address and client type for Backend to establish connections with the MMF
type FunctionConfig struct {
	Host                 string              `protobuf:"bytes,1,opt,name=host,proto3" json:"host,omitempty"`
	Port                 int32               `protobuf:"varint,2,opt,name=port,proto3" json:"port,omitempty"`
	Type                 FunctionConfig_Type `protobuf:"varint,3,opt,name=type,proto3,enum=openmatch.FunctionConfig_Type" json:"type,omitempty"`
	XXX_NoUnkeyedLiteral struct{}            `json:"-"`
	XXX_unrecognized     []byte              `json:"-"`
	XXX_sizecache        int32               `json:"-"`
}

func (m *FunctionConfig) Reset()         { *m = FunctionConfig{} }
func (m *FunctionConfig) String() string { return proto.CompactTextString(m) }
func (*FunctionConfig) ProtoMessage()    {}
func (*FunctionConfig) Descriptor() ([]byte, []int) {
	return fileDescriptor_8dab762378f455cd, []int{0}
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

func (m *FunctionConfig) GetHost() string {
	if m != nil {
		return m.Host
	}
	return ""
}

func (m *FunctionConfig) GetPort() int32 {
	if m != nil {
		return m.Port
	}
	return 0
}

func (m *FunctionConfig) GetType() FunctionConfig_Type {
	if m != nil {
		return m.Type
	}
	return FunctionConfig_GRPC
}

type FetchMatchesRequest struct {
	// A configuration for the MatchFunction server of this FetchMatches call.
	Config *FunctionConfig `protobuf:"bytes,1,opt,name=config,proto3" json:"config,omitempty"`
	// A MatchProfile that will be sent to the MatchFunction server of this FetchMatches call.
	Profile              *MatchProfile `protobuf:"bytes,2,opt,name=profile,proto3" json:"profile,omitempty"`
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
	XXX_unrecognized     []byte        `json:"-"`
	XXX_sizecache        int32         `json:"-"`
}

func (m *FetchMatchesRequest) Reset()         { *m = FetchMatchesRequest{} }
func (m *FetchMatchesRequest) String() string { return proto.CompactTextString(m) }
func (*FetchMatchesRequest) ProtoMessage()    {}
func (*FetchMatchesRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_8dab762378f455cd, []int{1}
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

func (m *FetchMatchesRequest) GetProfile() *MatchProfile {
	if m != nil {
		return m.Profile
	}
	return nil
}

type FetchMatchesResponse struct {
	// A Match generated by the user-defined MMF with the specified MatchProfiles.
	// A valid Match response will contain at least one ticket.
	Match                *Match   `protobuf:"bytes,1,opt,name=match,proto3" json:"match,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *FetchMatchesResponse) Reset()         { *m = FetchMatchesResponse{} }
func (m *FetchMatchesResponse) String() string { return proto.CompactTextString(m) }
func (*FetchMatchesResponse) ProtoMessage()    {}
func (*FetchMatchesResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_8dab762378f455cd, []int{2}
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

type ReleaseTicketsRequest struct {
	// TicketIds is a list of string representing Open Match generated Ids to be re-enabled for MMF querying
	// because they are no longer awaiting assignment from a previous match result
	TicketIds            []string `protobuf:"bytes,1,rep,name=ticket_ids,json=ticketIds,proto3" json:"ticket_ids,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ReleaseTicketsRequest) Reset()         { *m = ReleaseTicketsRequest{} }
func (m *ReleaseTicketsRequest) String() string { return proto.CompactTextString(m) }
func (*ReleaseTicketsRequest) ProtoMessage()    {}
func (*ReleaseTicketsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_8dab762378f455cd, []int{3}
}

func (m *ReleaseTicketsRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ReleaseTicketsRequest.Unmarshal(m, b)
}
func (m *ReleaseTicketsRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ReleaseTicketsRequest.Marshal(b, m, deterministic)
}
func (m *ReleaseTicketsRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ReleaseTicketsRequest.Merge(m, src)
}
func (m *ReleaseTicketsRequest) XXX_Size() int {
	return xxx_messageInfo_ReleaseTicketsRequest.Size(m)
}
func (m *ReleaseTicketsRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ReleaseTicketsRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ReleaseTicketsRequest proto.InternalMessageInfo

func (m *ReleaseTicketsRequest) GetTicketIds() []string {
	if m != nil {
		return m.TicketIds
	}
	return nil
}

type ReleaseTicketsResponse struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ReleaseTicketsResponse) Reset()         { *m = ReleaseTicketsResponse{} }
func (m *ReleaseTicketsResponse) String() string { return proto.CompactTextString(m) }
func (*ReleaseTicketsResponse) ProtoMessage()    {}
func (*ReleaseTicketsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_8dab762378f455cd, []int{4}
}

func (m *ReleaseTicketsResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ReleaseTicketsResponse.Unmarshal(m, b)
}
func (m *ReleaseTicketsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ReleaseTicketsResponse.Marshal(b, m, deterministic)
}
func (m *ReleaseTicketsResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ReleaseTicketsResponse.Merge(m, src)
}
func (m *ReleaseTicketsResponse) XXX_Size() int {
	return xxx_messageInfo_ReleaseTicketsResponse.Size(m)
}
func (m *ReleaseTicketsResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ReleaseTicketsResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ReleaseTicketsResponse proto.InternalMessageInfo

// AssignmentGroup contains an Assignment and the Tickets to which it should be applied.
type AssignmentGroup struct {
	// TicketIds is a list of strings representing Open Match generated Ids which apply to an Assignment.
	TicketIds []string `protobuf:"bytes,1,rep,name=ticket_ids,json=ticketIds,proto3" json:"ticket_ids,omitempty"`
	// An Assignment specifies game connection related information to be associated with the TicketIds.
	Assignment           *Assignment `protobuf:"bytes,2,opt,name=assignment,proto3" json:"assignment,omitempty"`
	XXX_NoUnkeyedLiteral struct{}    `json:"-"`
	XXX_unrecognized     []byte      `json:"-"`
	XXX_sizecache        int32       `json:"-"`
}

func (m *AssignmentGroup) Reset()         { *m = AssignmentGroup{} }
func (m *AssignmentGroup) String() string { return proto.CompactTextString(m) }
func (*AssignmentGroup) ProtoMessage()    {}
func (*AssignmentGroup) Descriptor() ([]byte, []int) {
	return fileDescriptor_8dab762378f455cd, []int{5}
}

func (m *AssignmentGroup) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AssignmentGroup.Unmarshal(m, b)
}
func (m *AssignmentGroup) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AssignmentGroup.Marshal(b, m, deterministic)
}
func (m *AssignmentGroup) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AssignmentGroup.Merge(m, src)
}
func (m *AssignmentGroup) XXX_Size() int {
	return xxx_messageInfo_AssignmentGroup.Size(m)
}
func (m *AssignmentGroup) XXX_DiscardUnknown() {
	xxx_messageInfo_AssignmentGroup.DiscardUnknown(m)
}

var xxx_messageInfo_AssignmentGroup proto.InternalMessageInfo

func (m *AssignmentGroup) GetTicketIds() []string {
	if m != nil {
		return m.TicketIds
	}
	return nil
}

func (m *AssignmentGroup) GetAssignment() *Assignment {
	if m != nil {
		return m.Assignment
	}
	return nil
}

// AssignmentFailure contains the id of the Ticket that failed the Assignment and the failure status.
type AssignmentFailure struct {
	TicketId             string                  `protobuf:"bytes,1,opt,name=ticket_id,json=ticketId,proto3" json:"ticket_id,omitempty"`
	Cause                AssignmentFailure_Cause `protobuf:"varint,2,opt,name=cause,proto3,enum=openmatch.AssignmentFailure_Cause" json:"cause,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                `json:"-"`
	XXX_unrecognized     []byte                  `json:"-"`
	XXX_sizecache        int32                   `json:"-"`
}

func (m *AssignmentFailure) Reset()         { *m = AssignmentFailure{} }
func (m *AssignmentFailure) String() string { return proto.CompactTextString(m) }
func (*AssignmentFailure) ProtoMessage()    {}
func (*AssignmentFailure) Descriptor() ([]byte, []int) {
	return fileDescriptor_8dab762378f455cd, []int{6}
}

func (m *AssignmentFailure) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AssignmentFailure.Unmarshal(m, b)
}
func (m *AssignmentFailure) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AssignmentFailure.Marshal(b, m, deterministic)
}
func (m *AssignmentFailure) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AssignmentFailure.Merge(m, src)
}
func (m *AssignmentFailure) XXX_Size() int {
	return xxx_messageInfo_AssignmentFailure.Size(m)
}
func (m *AssignmentFailure) XXX_DiscardUnknown() {
	xxx_messageInfo_AssignmentFailure.DiscardUnknown(m)
}

var xxx_messageInfo_AssignmentFailure proto.InternalMessageInfo

func (m *AssignmentFailure) GetTicketId() string {
	if m != nil {
		return m.TicketId
	}
	return ""
}

func (m *AssignmentFailure) GetCause() AssignmentFailure_Cause {
	if m != nil {
		return m.Cause
	}
	return AssignmentFailure_UNKNOWN
}

type AssignTicketsRequest struct {
	// Assignments is a list of assignment groups that contain assignment and the Tickets to which they should be applied.
	Assignments          []*AssignmentGroup `protobuf:"bytes,1,rep,name=assignments,proto3" json:"assignments,omitempty"`
	XXX_NoUnkeyedLiteral struct{}           `json:"-"`
	XXX_unrecognized     []byte             `json:"-"`
	XXX_sizecache        int32              `json:"-"`
}

func (m *AssignTicketsRequest) Reset()         { *m = AssignTicketsRequest{} }
func (m *AssignTicketsRequest) String() string { return proto.CompactTextString(m) }
func (*AssignTicketsRequest) ProtoMessage()    {}
func (*AssignTicketsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_8dab762378f455cd, []int{7}
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

func (m *AssignTicketsRequest) GetAssignments() []*AssignmentGroup {
	if m != nil {
		return m.Assignments
	}
	return nil
}

type AssignTicketsResponse struct {
	// Failures is a list of all the Tickets that failed assignment along with the cause of failure.
	Failures             []*AssignmentFailure `protobuf:"bytes,1,rep,name=failures,proto3" json:"failures,omitempty"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
}

func (m *AssignTicketsResponse) Reset()         { *m = AssignTicketsResponse{} }
func (m *AssignTicketsResponse) String() string { return proto.CompactTextString(m) }
func (*AssignTicketsResponse) ProtoMessage()    {}
func (*AssignTicketsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_8dab762378f455cd, []int{8}
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

func (m *AssignTicketsResponse) GetFailures() []*AssignmentFailure {
	if m != nil {
		return m.Failures
	}
	return nil
}

func init() {
	proto.RegisterEnum("openmatch.FunctionConfig_Type", FunctionConfig_Type_name, FunctionConfig_Type_value)
	proto.RegisterEnum("openmatch.AssignmentFailure_Cause", AssignmentFailure_Cause_name, AssignmentFailure_Cause_value)
	proto.RegisterType((*FunctionConfig)(nil), "openmatch.FunctionConfig")
	proto.RegisterType((*FetchMatchesRequest)(nil), "openmatch.FetchMatchesRequest")
	proto.RegisterType((*FetchMatchesResponse)(nil), "openmatch.FetchMatchesResponse")
	proto.RegisterType((*ReleaseTicketsRequest)(nil), "openmatch.ReleaseTicketsRequest")
	proto.RegisterType((*ReleaseTicketsResponse)(nil), "openmatch.ReleaseTicketsResponse")
	proto.RegisterType((*AssignmentGroup)(nil), "openmatch.AssignmentGroup")
	proto.RegisterType((*AssignmentFailure)(nil), "openmatch.AssignmentFailure")
	proto.RegisterType((*AssignTicketsRequest)(nil), "openmatch.AssignTicketsRequest")
	proto.RegisterType((*AssignTicketsResponse)(nil), "openmatch.AssignTicketsResponse")
}

func init() { proto.RegisterFile("api/backend.proto", fileDescriptor_8dab762378f455cd) }

var fileDescriptor_8dab762378f455cd = []byte{
	// 879 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x84, 0x55, 0xdd, 0x6e, 0x1b, 0x45,
	0x14, 0xee, 0xd8, 0xce, 0x8f, 0x8f, 0xc1, 0xb8, 0x43, 0x52, 0x8c, 0x29, 0x74, 0xb3, 0x88, 0x12,
	0x99, 0x7a, 0x37, 0x59, 0x02, 0xaa, 0xcc, 0x8f, 0x9a, 0xba, 0x49, 0x15, 0xb5, 0x38, 0x65, 0xe3,
	0x82, 0xc4, 0x4d, 0xb4, 0x5e, 0x1f, 0xaf, 0x97, 0xd8, 0x3b, 0xc3, 0xce, 0x6c, 0x4a, 0x85, 0x84,
	0x10, 0xe2, 0x02, 0x71, 0x09, 0x12, 0x17, 0x7d, 0x04, 0xee, 0x78, 0x16, 0x6e, 0x78, 0x00, 0x1e,
	0x04, 0xed, 0xcc, 0xfa, 0x37, 0x4e, 0x7a, 0xe5, 0x99, 0x39, 0xdf, 0xf9, 0xbe, 0xef, 0x9c, 0x3d,
	0x33, 0x86, 0xeb, 0x1e, 0x0f, 0xed, 0xae, 0xe7, 0x9f, 0x61, 0xd4, 0xb3, 0x78, 0xcc, 0x24, 0xa3,
	0x45, 0xc6, 0x31, 0x1a, 0x79, 0xd2, 0x1f, 0xd4, 0x68, 0x1a, 0x1d, 0xa1, 0x10, 0x5e, 0x80, 0x42,
	0x87, 0x6b, 0x37, 0x03, 0xc6, 0x82, 0x21, 0xda, 0x69, 0xc8, 0x8b, 0x22, 0x26, 0x3d, 0x19, 0xb2,
	0x68, 0x1c, 0xbd, 0xa3, 0x7e, 0xfc, 0x46, 0x80, 0x51, 0x43, 0x3c, 0xf3, 0x82, 0x00, 0x63, 0x9b,
	0x71, 0x85, 0xb8, 0x88, 0x36, 0x7f, 0x25, 0x50, 0x3e, 0x4c, 0x22, 0x3f, 0x3d, 0x6b, 0xb1, 0xa8,
	0x1f, 0x06, 0x94, 0x42, 0x61, 0xc0, 0x84, 0xac, 0x12, 0x83, 0x6c, 0x17, 0x5d, 0xb5, 0x4e, 0xcf,
	0x38, 0x8b, 0x65, 0x35, 0x67, 0x90, 0xed, 0x15, 0x57, 0xad, 0xa9, 0x03, 0x05, 0xf9, 0x9c, 0x63,
	0x35, 0x6f, 0x90, 0xed, 0xb2, 0xf3, 0x8e, 0x35, 0x31, 0x6d, 0xcd, 0x13, 0x5a, 0x9d, 0xe7, 0x1c,
	0x5d, 0x85, 0x35, 0x6b, 0x50, 0x48, 0x77, 0x74, 0x1d, 0x0a, 0x0f, 0xdd, 0x27, 0xad, 0xca, 0xb5,
	0x74, 0xe5, 0x1e, 0x9c, 0x74, 0x2a, 0xc4, 0xfc, 0x01, 0x5e, 0x3f, 0x44, 0xe9, 0x0f, 0xbe, 0x48,
	0x39, 0x50, 0xb8, 0xf8, 0x5d, 0x82, 0x42, 0xd2, 0x5d, 0x58, 0xf5, 0x15, 0x8f, 0x32, 0x54, 0x72,
	0xde, 0xbc, 0x54, 0xc8, 0xcd, 0x80, 0x74, 0x17, 0xd6, 0x78, 0xcc, 0xfa, 0xe1, 0x10, 0x95, 0xe1,
	0x92, 0xf3, 0xc6, 0x4c, 0x8e, 0xa2, 0x7f, 0xa2, 0xc3, 0xee, 0x18, 0x67, 0x7e, 0x0e, 0x1b, 0xf3,
	0xe2, 0x82, 0xb3, 0x48, 0x20, 0xbd, 0x0d, 0x2b, 0x2a, 0x2d, 0x13, 0xaf, 0x2c, 0x12, 0xb9, 0x3a,
	0x6c, 0x7e, 0x0c, 0x9b, 0x2e, 0x0e, 0xd1, 0x13, 0xd8, 0x09, 0xfd, 0x33, 0x94, 0x13, 0xfb, 0x6f,
	0x03, 0x48, 0x75, 0x72, 0x1a, 0xf6, 0x44, 0x95, 0x18, 0xf9, 0xed, 0xa2, 0x5b, 0xd4, 0x27, 0x47,
	0x3d, 0x61, 0x56, 0xe1, 0xc6, 0x62, 0x9e, 0x56, 0x36, 0x03, 0x78, 0x6d, 0x5f, 0x88, 0x30, 0x88,
	0x46, 0x18, 0xc9, 0x87, 0x31, 0x4b, 0xf8, 0x4b, 0xb8, 0xe8, 0x47, 0x00, 0xde, 0x24, 0x23, 0xab,
	0x7c, 0x73, 0xc6, 0xf0, 0x94, 0xce, 0x9d, 0x01, 0x9a, 0x7f, 0x12, 0xb8, 0x3e, 0x0d, 0x1d, 0x7a,
	0xe1, 0x30, 0x89, 0x91, 0xbe, 0x05, 0xc5, 0x89, 0x56, 0x36, 0x0a, 0xeb, 0x63, 0x29, 0x7a, 0x17,
	0x56, 0x7c, 0x2f, 0x11, 0xba, 0xbd, 0x65, 0xc7, 0x5c, 0x2a, 0x92, 0x31, 0x59, 0xad, 0x14, 0xe9,
	0xea, 0x04, 0xb3, 0x0e, 0x2b, 0x6a, 0x4f, 0x4b, 0xb0, 0xf6, 0xb4, 0xfd, 0xa8, 0x7d, 0xfc, 0x75,
	0xbb, 0x72, 0x8d, 0x6e, 0x40, 0xa5, 0x73, 0xd4, 0x7a, 0x74, 0xd0, 0x39, 0x6d, 0x1f, 0x77, 0x4e,
	0x0f, 0x8f, 0x9f, 0xb6, 0x1f, 0x54, 0x88, 0xd9, 0x81, 0x0d, 0xcd, 0xb6, 0xd0, 0xd2, 0x4f, 0xa1,
	0x34, 0xb5, 0xaf, 0xfb, 0x50, 0x72, 0x6a, 0x4b, 0x3d, 0xa8, 0xbe, 0xb9, 0xb3, 0x70, 0xf3, 0x4b,
	0xd8, 0x5c, 0x60, 0xcd, 0x3e, 0xf5, 0x5d, 0x58, 0xef, 0x6b, 0xcb, 0x63, 0xce, 0x9b, 0x57, 0xd5,
	0xe5, 0x4e, 0xd0, 0xce, 0x8b, 0x3c, 0x94, 0xef, 0xeb, 0x1b, 0x7c, 0x82, 0xf1, 0x79, 0xe8, 0x23,
	0xfd, 0x11, 0x5e, 0x99, 0x9d, 0x27, 0x3a, 0x77, 0x3d, 0x2e, 0x4e, 0x79, 0xed, 0xd6, 0xa5, 0xf1,
	0x6c, 0x1c, 0x3e, 0xf8, 0xf9, 0x9f, 0xff, 0xfe, 0xc8, 0xbd, 0x67, 0x1a, 0xf6, 0xf9, 0xee, 0xf8,
	0xb9, 0x10, 0x5a, 0xcc, 0x1e, 0x69, 0x6c, 0xb3, 0x9f, 0x26, 0x36, 0x49, 0x7d, 0x87, 0xd0, 0x9f,
	0x08, 0xbc, 0x3a, 0x57, 0x26, 0xbd, 0x75, 0xa1, 0x98, 0xf9, 0xb6, 0xd6, 0x8c, 0xcb, 0x01, 0x99,
	0x87, 0x3b, 0xca, 0xc3, 0x6d, 0x73, 0x6b, 0x89, 0x07, 0x3d, 0x1b, 0xa2, 0xa9, 0x5b, 0xdd, 0x24,
	0x75, 0xfa, 0x0b, 0x81, 0xf2, 0xfc, 0x6c, 0xd3, 0x59, 0x89, 0xa5, 0xd7, 0xa5, 0xb6, 0x75, 0x05,
	0x22, 0x73, 0xd1, 0x50, 0x2e, 0xde, 0x37, 0xcd, 0x2b, 0x5c, 0xc4, 0x3a, 0xb5, 0x49, 0xea, 0xf7,
	0x7f, 0xcb, 0xff, 0xbe, 0xff, 0x6f, 0x8e, 0xfe, 0x4d, 0x60, 0x2d, 0xfb, 0x46, 0xe6, 0x11, 0xc0,
	0x31, 0xc7, 0xc8, 0x50, 0x3d, 0xa6, 0x37, 0x06, 0x52, 0x72, 0xd1, 0xb4, 0xed, 0x54, 0xb9, 0xa1,
	0xa5, 0x7b, 0x78, 0x5e, 0x7b, 0x77, 0xba, 0x6f, 0xf4, 0x42, 0xe1, 0x27, 0x42, 0xdc, 0xd3, 0x2f,
	0x6f, 0x90, 0x4e, 0x95, 0xb0, 0x7c, 0x36, 0xaa, 0x7f, 0x05, 0x74, 0x9f, 0x7b, 0xfe, 0x00, 0x0d,
	0xc7, 0xda, 0x31, 0x1e, 0x87, 0x3e, 0xa6, 0xa3, 0x74, 0x6f, 0x4c, 0x19, 0x84, 0x72, 0x90, 0x74,
	0x53, 0xa4, 0xad, 0x53, 0xfb, 0x2c, 0x0e, 0xbc, 0x11, 0x8a, 0x19, 0x31, 0xbb, 0x3b, 0x64, 0x5d,
	0x7b, 0xe4, 0x09, 0x89, 0xb1, 0xfd, 0xf8, 0xa8, 0x75, 0xd0, 0x3e, 0x39, 0x70, 0xf2, 0xbb, 0xd6,
	0x4e, 0x3d, 0x47, 0x72, 0x4e, 0xc5, 0xe3, 0x7c, 0x18, 0xfa, 0xea, 0xd1, 0xb6, 0xbf, 0x15, 0x2c,
	0x6a, 0x5e, 0x38, 0x71, 0x3f, 0x81, 0xfc, 0xde, 0xce, 0x1e, 0xdd, 0x83, 0xba, 0x8b, 0x32, 0x89,
	0x23, 0xec, 0x19, 0xcf, 0x06, 0x18, 0x19, 0x72, 0x80, 0x46, 0x8c, 0x82, 0x25, 0xb1, 0x8f, 0x46,
	0x8f, 0xa1, 0x30, 0x22, 0x26, 0x0d, 0xfc, 0x3e, 0x14, 0xd2, 0xa2, 0xab, 0x50, 0x78, 0x91, 0x23,
	0x6b, 0xf1, 0x67, 0x50, 0x9d, 0x36, 0xc3, 0x78, 0xc0, 0xfc, 0x24, 0x1d, 0x72, 0xc5, 0x4e, 0xb7,
	0x96, 0xb7, 0xc6, 0x16, 0xa1, 0x44, 0xbb, 0xc7, 0x7c, 0x61, 0x7f, 0x63, 0x2c, 0x84, 0x66, 0xea,
	0xe2, 0x67, 0x81, 0xcd, 0xbb, 0x7f, 0xe5, 0x8a, 0x29, 0xbf, 0xa2, 0xef, 0xae, 0xaa, 0x7f, 0x9d,
	0x0f, 0xff, 0x0f, 0x00, 0x00, 0xff, 0xff, 0x9d, 0x6e, 0x5a, 0xc2, 0xf5, 0x06, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// BackendServiceClient is the client API for BackendService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type BackendServiceClient interface {
	// FetchMatches triggers a MatchFunction with the specified MatchProfile and returns a set of match proposals that
	// match the description of that MatchProfile.
	// FetchMatches immediately returns an error if it encounters any execution failures.
	FetchMatches(ctx context.Context, in *FetchMatchesRequest, opts ...grpc.CallOption) (BackendService_FetchMatchesClient, error)
	// AssignTickets overwrites the Assignment field of the input TicketIds.
	AssignTickets(ctx context.Context, in *AssignTicketsRequest, opts ...grpc.CallOption) (*AssignTicketsResponse, error)
	// ReleaseTickets removes the submitted tickets from the list that prevents tickets
	// that are awaiting assignment from appearing in MMF queries, effectively putting them back into
	// the matchmaking pool
	//
	// BETA FEATURE WARNING:  This call and the associated Request and Response
	// messages are not finalized and still subject to possible change or removal.
	ReleaseTickets(ctx context.Context, in *ReleaseTicketsRequest, opts ...grpc.CallOption) (*ReleaseTicketsResponse, error)
}

type backendServiceClient struct {
	cc *grpc.ClientConn
}

func NewBackendServiceClient(cc *grpc.ClientConn) BackendServiceClient {
	return &backendServiceClient{cc}
}

func (c *backendServiceClient) FetchMatches(ctx context.Context, in *FetchMatchesRequest, opts ...grpc.CallOption) (BackendService_FetchMatchesClient, error) {
	stream, err := c.cc.NewStream(ctx, &_BackendService_serviceDesc.Streams[0], "/openmatch.BackendService/FetchMatches", opts...)
	if err != nil {
		return nil, err
	}
	x := &backendServiceFetchMatchesClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type BackendService_FetchMatchesClient interface {
	Recv() (*FetchMatchesResponse, error)
	grpc.ClientStream
}

type backendServiceFetchMatchesClient struct {
	grpc.ClientStream
}

func (x *backendServiceFetchMatchesClient) Recv() (*FetchMatchesResponse, error) {
	m := new(FetchMatchesResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *backendServiceClient) AssignTickets(ctx context.Context, in *AssignTicketsRequest, opts ...grpc.CallOption) (*AssignTicketsResponse, error) {
	out := new(AssignTicketsResponse)
	err := c.cc.Invoke(ctx, "/openmatch.BackendService/AssignTickets", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *backendServiceClient) ReleaseTickets(ctx context.Context, in *ReleaseTicketsRequest, opts ...grpc.CallOption) (*ReleaseTicketsResponse, error) {
	out := new(ReleaseTicketsResponse)
	err := c.cc.Invoke(ctx, "/openmatch.BackendService/ReleaseTickets", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BackendServiceServer is the server API for BackendService service.
type BackendServiceServer interface {
	// FetchMatches triggers a MatchFunction with the specified MatchProfile and returns a set of match proposals that
	// match the description of that MatchProfile.
	// FetchMatches immediately returns an error if it encounters any execution failures.
	FetchMatches(*FetchMatchesRequest, BackendService_FetchMatchesServer) error
	// AssignTickets overwrites the Assignment field of the input TicketIds.
	AssignTickets(context.Context, *AssignTicketsRequest) (*AssignTicketsResponse, error)
	// ReleaseTickets removes the submitted tickets from the list that prevents tickets
	// that are awaiting assignment from appearing in MMF queries, effectively putting them back into
	// the matchmaking pool
	//
	// BETA FEATURE WARNING:  This call and the associated Request and Response
	// messages are not finalized and still subject to possible change or removal.
	ReleaseTickets(context.Context, *ReleaseTicketsRequest) (*ReleaseTicketsResponse, error)
}

// UnimplementedBackendServiceServer can be embedded to have forward compatible implementations.
type UnimplementedBackendServiceServer struct {
}

func (*UnimplementedBackendServiceServer) FetchMatches(req *FetchMatchesRequest, srv BackendService_FetchMatchesServer) error {
	return status.Errorf(codes.Unimplemented, "method FetchMatches not implemented")
}
func (*UnimplementedBackendServiceServer) AssignTickets(ctx context.Context, req *AssignTicketsRequest) (*AssignTicketsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AssignTickets not implemented")
}
func (*UnimplementedBackendServiceServer) ReleaseTickets(ctx context.Context, req *ReleaseTicketsRequest) (*ReleaseTicketsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReleaseTickets not implemented")
}

func RegisterBackendServiceServer(s *grpc.Server, srv BackendServiceServer) {
	s.RegisterService(&_BackendService_serviceDesc, srv)
}

func _BackendService_FetchMatches_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(FetchMatchesRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(BackendServiceServer).FetchMatches(m, &backendServiceFetchMatchesServer{stream})
}

type BackendService_FetchMatchesServer interface {
	Send(*FetchMatchesResponse) error
	grpc.ServerStream
}

type backendServiceFetchMatchesServer struct {
	grpc.ServerStream
}

func (x *backendServiceFetchMatchesServer) Send(m *FetchMatchesResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _BackendService_AssignTickets_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AssignTicketsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BackendServiceServer).AssignTickets(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/openmatch.BackendService/AssignTickets",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BackendServiceServer).AssignTickets(ctx, req.(*AssignTicketsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BackendService_ReleaseTickets_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReleaseTicketsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BackendServiceServer).ReleaseTickets(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/openmatch.BackendService/ReleaseTickets",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BackendServiceServer).ReleaseTickets(ctx, req.(*ReleaseTicketsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _BackendService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "openmatch.BackendService",
	HandlerType: (*BackendServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AssignTickets",
			Handler:    _BackendService_AssignTickets_Handler,
		},
		{
			MethodName: "ReleaseTickets",
			Handler:    _BackendService_ReleaseTickets_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "FetchMatches",
			Handler:       _BackendService_FetchMatches_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "api/backend.proto",
}
