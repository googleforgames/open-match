// Code generated by protoc-gen-go. DO NOT EDIT.
// source: api/messages.proto

package pb

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	any "github.com/golang/protobuf/ptypes/any"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	_ "google.golang.org/genproto/googleapis/rpc/status"
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

// A Ticket is a basic matchmaking entity in Open Match. A Ticket may represent
// an individual 'Player', a 'Group' of players, or any other concepts unique to
// your use case. Open Match will not interpret what the Ticket represents but
// just treat it as a matchmaking unit with a set of SearchFields. Open Match
// stores the Ticket in state storage and enables an Assignment to be set on the
// Ticket.
type Ticket struct {
	// Id represents an auto-generated Id issued by Open Match.
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	// An Assignment represents a game server assignment associated with a Ticket,
	// or whatever finalized matched state means for your use case.
	// Open Match does not require or inspect any fields on Assignment.
	Assignment *Assignment `protobuf:"bytes,3,opt,name=assignment,proto3" json:"assignment,omitempty"`
	// Search fields are the fields which Open Match is aware of, and can be used
	// when specifying filters.
	SearchFields *SearchFields `protobuf:"bytes,4,opt,name=search_fields,json=searchFields,proto3" json:"search_fields,omitempty"`
	// Customized information not inspected by Open Match, to be used by the match
	// making function, evaluator, and components making calls to Open Match.
	// Optional, depending on the requirements of the connected systems.
	Extensions map[string]*any.Any `protobuf:"bytes,5,rep,name=extensions,proto3" json:"extensions,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// Create time is the time the Ticket was created. It is populated by Open
	// Match at the time of Ticket creation.
	CreateTime           *timestamp.Timestamp `protobuf:"bytes,6,opt,name=create_time,json=createTime,proto3" json:"create_time,omitempty"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
}

func (m *Ticket) Reset()         { *m = Ticket{} }
func (m *Ticket) String() string { return proto.CompactTextString(m) }
func (*Ticket) ProtoMessage()    {}
func (*Ticket) Descriptor() ([]byte, []int) {
	return fileDescriptor_cb9fb1f207fd5b8c, []int{0}
}

func (m *Ticket) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Ticket.Unmarshal(m, b)
}
func (m *Ticket) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Ticket.Marshal(b, m, deterministic)
}
func (m *Ticket) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Ticket.Merge(m, src)
}
func (m *Ticket) XXX_Size() int {
	return xxx_messageInfo_Ticket.Size(m)
}
func (m *Ticket) XXX_DiscardUnknown() {
	xxx_messageInfo_Ticket.DiscardUnknown(m)
}

var xxx_messageInfo_Ticket proto.InternalMessageInfo

func (m *Ticket) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *Ticket) GetAssignment() *Assignment {
	if m != nil {
		return m.Assignment
	}
	return nil
}

func (m *Ticket) GetSearchFields() *SearchFields {
	if m != nil {
		return m.SearchFields
	}
	return nil
}

func (m *Ticket) GetExtensions() map[string]*any.Any {
	if m != nil {
		return m.Extensions
	}
	return nil
}

func (m *Ticket) GetCreateTime() *timestamp.Timestamp {
	if m != nil {
		return m.CreateTime
	}
	return nil
}

// Search fields are the fields which Open Match is aware of, and can be used
// when specifying filters.
type SearchFields struct {
	// Float arguments.  Filterable on ranges.
	DoubleArgs map[string]float64 `protobuf:"bytes,1,rep,name=double_args,json=doubleArgs,proto3" json:"double_args,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"fixed64,2,opt,name=value,proto3"`
	// String arguments.  Filterable on equality.
	StringArgs map[string]string `protobuf:"bytes,2,rep,name=string_args,json=stringArgs,proto3" json:"string_args,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// Filterable on presence or absence of given value.
	Tags                 []string `protobuf:"bytes,3,rep,name=tags,proto3" json:"tags,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SearchFields) Reset()         { *m = SearchFields{} }
func (m *SearchFields) String() string { return proto.CompactTextString(m) }
func (*SearchFields) ProtoMessage()    {}
func (*SearchFields) Descriptor() ([]byte, []int) {
	return fileDescriptor_cb9fb1f207fd5b8c, []int{1}
}

func (m *SearchFields) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SearchFields.Unmarshal(m, b)
}
func (m *SearchFields) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SearchFields.Marshal(b, m, deterministic)
}
func (m *SearchFields) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SearchFields.Merge(m, src)
}
func (m *SearchFields) XXX_Size() int {
	return xxx_messageInfo_SearchFields.Size(m)
}
func (m *SearchFields) XXX_DiscardUnknown() {
	xxx_messageInfo_SearchFields.DiscardUnknown(m)
}

var xxx_messageInfo_SearchFields proto.InternalMessageInfo

func (m *SearchFields) GetDoubleArgs() map[string]float64 {
	if m != nil {
		return m.DoubleArgs
	}
	return nil
}

func (m *SearchFields) GetStringArgs() map[string]string {
	if m != nil {
		return m.StringArgs
	}
	return nil
}

func (m *SearchFields) GetTags() []string {
	if m != nil {
		return m.Tags
	}
	return nil
}

// An Assignment represents a game server assignment associated with a Ticket.
// Open Match does not require or inspect any fields on assignment.
type Assignment struct {
	// Connection information for this Assignment.
	Connection string `protobuf:"bytes,1,opt,name=connection,proto3" json:"connection,omitempty"`
	// Customized information not inspected by Open Match, to be used by the match
	// making function, evaluator, and components making calls to Open Match.
	// Optional, depending on the requirements of the connected systems.
	Extensions           map[string]*any.Any `protobuf:"bytes,4,rep,name=extensions,proto3" json:"extensions,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}            `json:"-"`
	XXX_unrecognized     []byte              `json:"-"`
	XXX_sizecache        int32               `json:"-"`
}

func (m *Assignment) Reset()         { *m = Assignment{} }
func (m *Assignment) String() string { return proto.CompactTextString(m) }
func (*Assignment) ProtoMessage()    {}
func (*Assignment) Descriptor() ([]byte, []int) {
	return fileDescriptor_cb9fb1f207fd5b8c, []int{2}
}

func (m *Assignment) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Assignment.Unmarshal(m, b)
}
func (m *Assignment) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Assignment.Marshal(b, m, deterministic)
}
func (m *Assignment) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Assignment.Merge(m, src)
}
func (m *Assignment) XXX_Size() int {
	return xxx_messageInfo_Assignment.Size(m)
}
func (m *Assignment) XXX_DiscardUnknown() {
	xxx_messageInfo_Assignment.DiscardUnknown(m)
}

var xxx_messageInfo_Assignment proto.InternalMessageInfo

func (m *Assignment) GetConnection() string {
	if m != nil {
		return m.Connection
	}
	return ""
}

func (m *Assignment) GetExtensions() map[string]*any.Any {
	if m != nil {
		return m.Extensions
	}
	return nil
}

// Filters numerical values to only those within a range.
//   double_arg: "foo"
//   max: 10
//   min: 5
// matches:
//   {"foo": 5}
//   {"foo": 7.5}
//   {"foo": 10}
// does not match:
//   {"foo": 4}
//   {"foo": 10.01}
//   {"foo": "7.5"}
//   {}
type DoubleRangeFilter struct {
	// Name of the ticket's search_fields.double_args this Filter operates on.
	DoubleArg string `protobuf:"bytes,1,opt,name=double_arg,json=doubleArg,proto3" json:"double_arg,omitempty"`
	// Maximum value.
	Max float64 `protobuf:"fixed64,2,opt,name=max,proto3" json:"max,omitempty"`
	// Minimum value.
	Min                  float64  `protobuf:"fixed64,3,opt,name=min,proto3" json:"min,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DoubleRangeFilter) Reset()         { *m = DoubleRangeFilter{} }
func (m *DoubleRangeFilter) String() string { return proto.CompactTextString(m) }
func (*DoubleRangeFilter) ProtoMessage()    {}
func (*DoubleRangeFilter) Descriptor() ([]byte, []int) {
	return fileDescriptor_cb9fb1f207fd5b8c, []int{3}
}

func (m *DoubleRangeFilter) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DoubleRangeFilter.Unmarshal(m, b)
}
func (m *DoubleRangeFilter) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DoubleRangeFilter.Marshal(b, m, deterministic)
}
func (m *DoubleRangeFilter) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DoubleRangeFilter.Merge(m, src)
}
func (m *DoubleRangeFilter) XXX_Size() int {
	return xxx_messageInfo_DoubleRangeFilter.Size(m)
}
func (m *DoubleRangeFilter) XXX_DiscardUnknown() {
	xxx_messageInfo_DoubleRangeFilter.DiscardUnknown(m)
}

var xxx_messageInfo_DoubleRangeFilter proto.InternalMessageInfo

func (m *DoubleRangeFilter) GetDoubleArg() string {
	if m != nil {
		return m.DoubleArg
	}
	return ""
}

func (m *DoubleRangeFilter) GetMax() float64 {
	if m != nil {
		return m.Max
	}
	return 0
}

func (m *DoubleRangeFilter) GetMin() float64 {
	if m != nil {
		return m.Min
	}
	return 0
}

// Filters strings exactly equaling a value.
//   string_arg: "foo"
//   value: "bar"
// matches:
//   {"foo": "bar"}
// does not match:
//   {"foo": "baz"}
//   {"bar": "foo"}
//   {}
type StringEqualsFilter struct {
	// Name of the ticket's search_fields.string_args this Filter operates on.
	StringArg            string   `protobuf:"bytes,1,opt,name=string_arg,json=stringArg,proto3" json:"string_arg,omitempty"`
	Value                string   `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *StringEqualsFilter) Reset()         { *m = StringEqualsFilter{} }
func (m *StringEqualsFilter) String() string { return proto.CompactTextString(m) }
func (*StringEqualsFilter) ProtoMessage()    {}
func (*StringEqualsFilter) Descriptor() ([]byte, []int) {
	return fileDescriptor_cb9fb1f207fd5b8c, []int{4}
}

func (m *StringEqualsFilter) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_StringEqualsFilter.Unmarshal(m, b)
}
func (m *StringEqualsFilter) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_StringEqualsFilter.Marshal(b, m, deterministic)
}
func (m *StringEqualsFilter) XXX_Merge(src proto.Message) {
	xxx_messageInfo_StringEqualsFilter.Merge(m, src)
}
func (m *StringEqualsFilter) XXX_Size() int {
	return xxx_messageInfo_StringEqualsFilter.Size(m)
}
func (m *StringEqualsFilter) XXX_DiscardUnknown() {
	xxx_messageInfo_StringEqualsFilter.DiscardUnknown(m)
}

var xxx_messageInfo_StringEqualsFilter proto.InternalMessageInfo

func (m *StringEqualsFilter) GetStringArg() string {
	if m != nil {
		return m.StringArg
	}
	return ""
}

func (m *StringEqualsFilter) GetValue() string {
	if m != nil {
		return m.Value
	}
	return ""
}

// Filters to the tag being present on the search_fields.
//   tag: "foo"
// matches:
//   ["foo"]
//   ["bar","foo"]
// does not match:
//   ["bar"]
//   []
type TagPresentFilter struct {
	Tag                  string   `protobuf:"bytes,1,opt,name=tag,proto3" json:"tag,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *TagPresentFilter) Reset()         { *m = TagPresentFilter{} }
func (m *TagPresentFilter) String() string { return proto.CompactTextString(m) }
func (*TagPresentFilter) ProtoMessage()    {}
func (*TagPresentFilter) Descriptor() ([]byte, []int) {
	return fileDescriptor_cb9fb1f207fd5b8c, []int{5}
}

func (m *TagPresentFilter) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TagPresentFilter.Unmarshal(m, b)
}
func (m *TagPresentFilter) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TagPresentFilter.Marshal(b, m, deterministic)
}
func (m *TagPresentFilter) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TagPresentFilter.Merge(m, src)
}
func (m *TagPresentFilter) XXX_Size() int {
	return xxx_messageInfo_TagPresentFilter.Size(m)
}
func (m *TagPresentFilter) XXX_DiscardUnknown() {
	xxx_messageInfo_TagPresentFilter.DiscardUnknown(m)
}

var xxx_messageInfo_TagPresentFilter proto.InternalMessageInfo

func (m *TagPresentFilter) GetTag() string {
	if m != nil {
		return m.Tag
	}
	return ""
}

// Pool specfies a set of criteria that are used to select a subset of Tickets
// that meet all the criteria.
type Pool struct {
	// A developer-chosen human-readable name for this Pool.
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// Set of Filters indicating the filtering criteria. Selected tickets must
	// match every Filter.
	DoubleRangeFilters  []*DoubleRangeFilter  `protobuf:"bytes,2,rep,name=double_range_filters,json=doubleRangeFilters,proto3" json:"double_range_filters,omitempty"`
	StringEqualsFilters []*StringEqualsFilter `protobuf:"bytes,4,rep,name=string_equals_filters,json=stringEqualsFilters,proto3" json:"string_equals_filters,omitempty"`
	TagPresentFilters   []*TagPresentFilter   `protobuf:"bytes,5,rep,name=tag_present_filters,json=tagPresentFilters,proto3" json:"tag_present_filters,omitempty"`
	// If specified, only Tickets created before the specified time are selected.
	CreatedBefore *timestamp.Timestamp `protobuf:"bytes,6,opt,name=created_before,json=createdBefore,proto3" json:"created_before,omitempty"`
	// If specified, only Tickets created after the specified time are selected.
	CreatedAfter         *timestamp.Timestamp `protobuf:"bytes,7,opt,name=created_after,json=createdAfter,proto3" json:"created_after,omitempty"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
}

func (m *Pool) Reset()         { *m = Pool{} }
func (m *Pool) String() string { return proto.CompactTextString(m) }
func (*Pool) ProtoMessage()    {}
func (*Pool) Descriptor() ([]byte, []int) {
	return fileDescriptor_cb9fb1f207fd5b8c, []int{6}
}

func (m *Pool) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Pool.Unmarshal(m, b)
}
func (m *Pool) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Pool.Marshal(b, m, deterministic)
}
func (m *Pool) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Pool.Merge(m, src)
}
func (m *Pool) XXX_Size() int {
	return xxx_messageInfo_Pool.Size(m)
}
func (m *Pool) XXX_DiscardUnknown() {
	xxx_messageInfo_Pool.DiscardUnknown(m)
}

var xxx_messageInfo_Pool proto.InternalMessageInfo

func (m *Pool) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Pool) GetDoubleRangeFilters() []*DoubleRangeFilter {
	if m != nil {
		return m.DoubleRangeFilters
	}
	return nil
}

func (m *Pool) GetStringEqualsFilters() []*StringEqualsFilter {
	if m != nil {
		return m.StringEqualsFilters
	}
	return nil
}

func (m *Pool) GetTagPresentFilters() []*TagPresentFilter {
	if m != nil {
		return m.TagPresentFilters
	}
	return nil
}

func (m *Pool) GetCreatedBefore() *timestamp.Timestamp {
	if m != nil {
		return m.CreatedBefore
	}
	return nil
}

func (m *Pool) GetCreatedAfter() *timestamp.Timestamp {
	if m != nil {
		return m.CreatedAfter
	}
	return nil
}

// A MatchProfile is Open Match's representation of a Match specification. It is
// used to indicate the criteria for selecting players for a match. A
// MatchProfile is the input to the API to get matches and is passed to the
// MatchFunction. It contains all the information required by the MatchFunction
// to generate match proposals.
type MatchProfile struct {
	// Name of this match profile.
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// Set of pools to be queried when generating a match for this MatchProfile.
	Pools []*Pool `protobuf:"bytes,3,rep,name=pools,proto3" json:"pools,omitempty"`
	// Customized information not inspected by Open Match, to be used by the match
	// making function, evaluator, and components making calls to Open Match.
	// Optional, depending on the requirements of the connected systems.
	Extensions           map[string]*any.Any `protobuf:"bytes,5,rep,name=extensions,proto3" json:"extensions,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}            `json:"-"`
	XXX_unrecognized     []byte              `json:"-"`
	XXX_sizecache        int32               `json:"-"`
}

func (m *MatchProfile) Reset()         { *m = MatchProfile{} }
func (m *MatchProfile) String() string { return proto.CompactTextString(m) }
func (*MatchProfile) ProtoMessage()    {}
func (*MatchProfile) Descriptor() ([]byte, []int) {
	return fileDescriptor_cb9fb1f207fd5b8c, []int{7}
}

func (m *MatchProfile) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_MatchProfile.Unmarshal(m, b)
}
func (m *MatchProfile) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_MatchProfile.Marshal(b, m, deterministic)
}
func (m *MatchProfile) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MatchProfile.Merge(m, src)
}
func (m *MatchProfile) XXX_Size() int {
	return xxx_messageInfo_MatchProfile.Size(m)
}
func (m *MatchProfile) XXX_DiscardUnknown() {
	xxx_messageInfo_MatchProfile.DiscardUnknown(m)
}

var xxx_messageInfo_MatchProfile proto.InternalMessageInfo

func (m *MatchProfile) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *MatchProfile) GetPools() []*Pool {
	if m != nil {
		return m.Pools
	}
	return nil
}

func (m *MatchProfile) GetExtensions() map[string]*any.Any {
	if m != nil {
		return m.Extensions
	}
	return nil
}

// A Match is used to represent a completed match object. It can be generated by
// a MatchFunction as a proposal or can be returned by OpenMatch as a result in
// response to the FetchMatches call.
// When a match is returned by the FetchMatches call, it should contain at least
// one ticket to be considered as valid.
type Match struct {
	// A Match ID that should be passed through the stack for tracing.
	MatchId string `protobuf:"bytes,1,opt,name=match_id,json=matchId,proto3" json:"match_id,omitempty"`
	// Name of the match profile that generated this Match.
	MatchProfile string `protobuf:"bytes,2,opt,name=match_profile,json=matchProfile,proto3" json:"match_profile,omitempty"`
	// Name of the match function that generated this Match.
	MatchFunction string `protobuf:"bytes,3,opt,name=match_function,json=matchFunction,proto3" json:"match_function,omitempty"`
	// Tickets belonging to this match.
	Tickets []*Ticket `protobuf:"bytes,4,rep,name=tickets,proto3" json:"tickets,omitempty"`
	// Customized information not inspected by Open Match, to be used by the match
	// making function, evaluator, and components making calls to Open Match.
	// Optional, depending on the requirements of the connected systems.
	Extensions           map[string]*any.Any `protobuf:"bytes,7,rep,name=extensions,proto3" json:"extensions,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}            `json:"-"`
	XXX_unrecognized     []byte              `json:"-"`
	XXX_sizecache        int32               `json:"-"`
}

func (m *Match) Reset()         { *m = Match{} }
func (m *Match) String() string { return proto.CompactTextString(m) }
func (*Match) ProtoMessage()    {}
func (*Match) Descriptor() ([]byte, []int) {
	return fileDescriptor_cb9fb1f207fd5b8c, []int{8}
}

func (m *Match) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Match.Unmarshal(m, b)
}
func (m *Match) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Match.Marshal(b, m, deterministic)
}
func (m *Match) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Match.Merge(m, src)
}
func (m *Match) XXX_Size() int {
	return xxx_messageInfo_Match.Size(m)
}
func (m *Match) XXX_DiscardUnknown() {
	xxx_messageInfo_Match.DiscardUnknown(m)
}

var xxx_messageInfo_Match proto.InternalMessageInfo

func (m *Match) GetMatchId() string {
	if m != nil {
		return m.MatchId
	}
	return ""
}

func (m *Match) GetMatchProfile() string {
	if m != nil {
		return m.MatchProfile
	}
	return ""
}

func (m *Match) GetMatchFunction() string {
	if m != nil {
		return m.MatchFunction
	}
	return ""
}

func (m *Match) GetTickets() []*Ticket {
	if m != nil {
		return m.Tickets
	}
	return nil
}

func (m *Match) GetExtensions() map[string]*any.Any {
	if m != nil {
		return m.Extensions
	}
	return nil
}

func init() {
	proto.RegisterType((*Ticket)(nil), "openmatch.Ticket")
	proto.RegisterMapType((map[string]*any.Any)(nil), "openmatch.Ticket.ExtensionsEntry")
	proto.RegisterType((*SearchFields)(nil), "openmatch.SearchFields")
	proto.RegisterMapType((map[string]float64)(nil), "openmatch.SearchFields.DoubleArgsEntry")
	proto.RegisterMapType((map[string]string)(nil), "openmatch.SearchFields.StringArgsEntry")
	proto.RegisterType((*Assignment)(nil), "openmatch.Assignment")
	proto.RegisterMapType((map[string]*any.Any)(nil), "openmatch.Assignment.ExtensionsEntry")
	proto.RegisterType((*DoubleRangeFilter)(nil), "openmatch.DoubleRangeFilter")
	proto.RegisterType((*StringEqualsFilter)(nil), "openmatch.StringEqualsFilter")
	proto.RegisterType((*TagPresentFilter)(nil), "openmatch.TagPresentFilter")
	proto.RegisterType((*Pool)(nil), "openmatch.Pool")
	proto.RegisterType((*MatchProfile)(nil), "openmatch.MatchProfile")
	proto.RegisterMapType((map[string]*any.Any)(nil), "openmatch.MatchProfile.ExtensionsEntry")
	proto.RegisterType((*Match)(nil), "openmatch.Match")
	proto.RegisterMapType((map[string]*any.Any)(nil), "openmatch.Match.ExtensionsEntry")
}

func init() { proto.RegisterFile("api/messages.proto", fileDescriptor_cb9fb1f207fd5b8c) }

var fileDescriptor_cb9fb1f207fd5b8c = []byte{
	// 830 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xc4, 0x55, 0xdd, 0x6e, 0xe3, 0x44,
	0x14, 0x56, 0x6c, 0x27, 0x69, 0x4e, 0xd2, 0xd6, 0x9d, 0xed, 0x6a, 0xbd, 0x81, 0x85, 0x60, 0xa8,
	0xa8, 0x40, 0x38, 0x52, 0x11, 0x12, 0xe2, 0x47, 0x90, 0x15, 0x2d, 0x6c, 0x11, 0x50, 0xdc, 0x8a,
	0x0b, 0x6e, 0xac, 0x49, 0x3c, 0xf1, 0x5a, 0xb5, 0xc7, 0xc6, 0x33, 0x59, 0x6d, 0xde, 0x83, 0xa7,
	0xe0, 0x9a, 0x6b, 0x9e, 0x82, 0x87, 0xe0, 0x9e, 0x17, 0x40, 0xf3, 0x63, 0x67, 0xd6, 0x09, 0xbb,
	0xdc, 0xa0, 0xde, 0xcd, 0x9c, 0x9f, 0x6f, 0xce, 0xf9, 0xce, 0xe7, 0x63, 0x40, 0xb8, 0x4c, 0xa7,
	0x39, 0x61, 0x0c, 0x27, 0x84, 0x05, 0x65, 0x55, 0xf0, 0x02, 0x0d, 0x8a, 0x92, 0xd0, 0x1c, 0xf3,
	0xc5, 0xd3, 0xf1, 0x83, 0xa4, 0x28, 0x92, 0x8c, 0x4c, 0xab, 0x72, 0x31, 0x65, 0x1c, 0xf3, 0x95,
	0x8e, 0x19, 0x3f, 0xd4, 0x0e, 0x79, 0x9b, 0xaf, 0x96, 0x53, 0x4c, 0xd7, 0xda, 0xf5, 0x66, 0xdb,
	0xc5, 0xd3, 0x9c, 0x30, 0x8e, 0xf3, 0x52, 0x05, 0xf8, 0x7f, 0x59, 0xd0, 0xbb, 0x49, 0x17, 0xb7,
	0x84, 0xa3, 0x03, 0xb0, 0xd2, 0xd8, 0xeb, 0x4c, 0x3a, 0xa7, 0x83, 0xd0, 0x4a, 0x63, 0xf4, 0x11,
	0x00, 0x66, 0x2c, 0x4d, 0x68, 0x4e, 0x28, 0xf7, 0xec, 0x49, 0xe7, 0x74, 0x78, 0x76, 0x3f, 0x68,
	0xea, 0x09, 0x66, 0x8d, 0x33, 0x34, 0x02, 0xd1, 0x67, 0xb0, 0xcf, 0x08, 0xae, 0x16, 0x4f, 0xa3,
	0x65, 0x4a, 0xb2, 0x98, 0x79, 0x8e, 0xcc, 0x7c, 0x60, 0x64, 0x5e, 0x4b, 0xff, 0x85, 0x74, 0x87,
	0x23, 0x66, 0xdc, 0xd0, 0x0c, 0x80, 0x3c, 0xe7, 0x84, 0xb2, 0xb4, 0xa0, 0xcc, 0xeb, 0x4e, 0xec,
	0xd3, 0xe1, 0xd9, 0x5b, 0x46, 0xaa, 0xaa, 0x35, 0x38, 0x6f, 0x62, 0xce, 0x29, 0xaf, 0xd6, 0xa1,
	0x91, 0x84, 0x3e, 0x85, 0xe1, 0xa2, 0x22, 0x98, 0x93, 0x48, 0x34, 0xeb, 0xf5, 0xe4, 0xf3, 0xe3,
	0x40, 0x31, 0x11, 0xd4, 0x4c, 0x04, 0x37, 0x35, 0x13, 0x21, 0xa8, 0x70, 0x61, 0x18, 0x5f, 0xc3,
	0x61, 0x0b, 0x1b, 0xb9, 0x60, 0xdf, 0x92, 0xb5, 0x26, 0x46, 0x1c, 0xd1, 0x7b, 0xd0, 0x7d, 0x86,
	0xb3, 0x15, 0xf1, 0x2c, 0x89, 0x7d, 0xbc, 0x85, 0x3d, 0xa3, 0xeb, 0x50, 0x85, 0x7c, 0x62, 0x7d,
	0xdc, 0xb9, 0x74, 0xf6, 0x2c, 0xd7, 0xf6, 0x7f, 0xb7, 0x60, 0x64, 0x76, 0x8e, 0xbe, 0x81, 0x61,
	0x5c, 0xac, 0xe6, 0x19, 0x89, 0x70, 0x95, 0x30, 0xaf, 0x23, 0x9b, 0x7d, 0xf7, 0x5f, 0x78, 0x0a,
	0xbe, 0x92, 0xa1, 0xb3, 0x2a, 0xa9, 0x5b, 0x8e, 0x1b, 0x83, 0x40, 0x62, 0xbc, 0x4a, 0x69, 0xa2,
	0x90, 0xac, 0x97, 0x23, 0x5d, 0xcb, 0x50, 0x03, 0x89, 0x35, 0x06, 0x84, 0xc0, 0xe1, 0x38, 0x61,
	0x9e, 0x3d, 0xb1, 0x4f, 0x07, 0xa1, 0x3c, 0x8f, 0x3f, 0x87, 0xc3, 0xd6, 0xe3, 0x3b, 0x38, 0x39,
	0x36, 0x39, 0xe9, 0x18, 0xdd, 0x8b, 0xf4, 0xd6, 0x8b, 0xaf, 0x4a, 0x1f, 0x18, 0xe9, 0xfe, 0x9f,
	0x1d, 0x80, 0x8d, 0xd4, 0xd0, 0x1b, 0x00, 0x8b, 0x82, 0x52, 0xb2, 0xe0, 0x69, 0x41, 0x35, 0x82,
	0x61, 0x41, 0xe7, 0x2f, 0x08, 0xc8, 0x91, 0x4c, 0x9c, 0xec, 0x54, 0xed, 0xcb, 0x44, 0xf4, 0x3f,
	0xea, 0xe0, 0xd2, 0xd9, 0xb3, 0x5d, 0xc7, 0xff, 0x09, 0x8e, 0x14, 0xa9, 0x21, 0xa6, 0x09, 0xb9,
	0x48, 0x33, 0x4e, 0x2a, 0xf4, 0x08, 0x60, 0xa3, 0x08, 0xfd, 0xd2, 0xa0, 0x99, 0xb3, 0xa8, 0x20,
	0xc7, 0xcf, 0x35, 0xc3, 0xe2, 0x28, 0x2d, 0x29, 0x95, 0x1f, 0xa7, 0xb0, 0xa4, 0xd4, 0x7f, 0x02,
	0x48, 0xb1, 0x7d, 0xfe, 0xcb, 0x0a, 0x67, 0x6c, 0x03, 0xbc, 0x11, 0x48, 0x0d, 0xdc, 0x8c, 0x7d,
	0x37, 0xfb, 0xfe, 0x3b, 0xe0, 0xde, 0xe0, 0xe4, 0xaa, 0x22, 0x8c, 0x50, 0xae, 0x81, 0x5c, 0xb0,
	0x39, 0xae, 0x11, 0xc4, 0xd1, 0xff, 0xd5, 0x06, 0xe7, 0xaa, 0x28, 0x32, 0x21, 0x1d, 0x8a, 0x73,
	0xa2, 0x7d, 0xf2, 0x8c, 0xbe, 0x87, 0x63, 0xdd, 0x50, 0x25, 0xda, 0x8c, 0x96, 0x12, 0xa5, 0x56,
	0xe8, 0xeb, 0xc6, 0x5c, 0xb6, 0xc8, 0x08, 0x51, 0xdc, 0x36, 0x31, 0xf4, 0x23, 0xdc, 0xd7, 0x7d,
	0x10, 0xd9, 0x5e, 0x03, 0xa8, 0x06, 0xfd, 0xc8, 0x94, 0xfc, 0x16, 0x0b, 0xe1, 0x3d, 0xb6, 0x65,
	0x63, 0xe8, 0x5b, 0xb8, 0xc7, 0x71, 0x12, 0x95, 0xaa, 0xcd, 0x06, 0x50, 0xad, 0x9e, 0xd7, 0xcc,
	0xd5, 0xd3, 0xe2, 0x22, 0x3c, 0xe2, 0x2d, 0x8b, 0x58, 0x5f, 0x07, 0x6a, 0x99, 0xc4, 0xd1, 0x9c,
	0x2c, 0x8b, 0xea, 0xbf, 0xac, 0x9f, 0x7d, 0x9d, 0xf1, 0x58, 0x26, 0xa0, 0x2f, 0xa0, 0x36, 0x44,
	0x78, 0xc9, 0x49, 0xe5, 0xf5, 0x5f, 0x89, 0x30, 0xd2, 0x09, 0x33, 0x11, 0xaf, 0xf5, 0xf5, 0x77,
	0x07, 0x46, 0xdf, 0x89, 0xba, 0xaf, 0xaa, 0x62, 0x99, 0x66, 0x64, 0xe7, 0x78, 0x4e, 0xa0, 0x5b,
	0x16, 0x45, 0xa6, 0x3e, 0xf7, 0xe1, 0xd9, 0xa1, 0xd1, 0xad, 0x18, 0x69, 0xa8, 0xbc, 0xe8, 0xeb,
	0x1d, 0x4b, 0xd9, 0xdc, 0x2e, 0xe6, 0x3b, 0x77, 0xf7, 0x55, 0x39, 0x6e, 0xd7, 0xff, 0xc3, 0x82,
	0xae, 0xac, 0x06, 0x3d, 0x84, 0x3d, 0x59, 0x5c, 0xd4, 0xfc, 0xd3, 0xfa, 0xf2, 0xfe, 0x24, 0x46,
	0x6f, 0xc3, 0xbe, 0x72, 0x95, 0xaa, 0x64, 0xad, 0xfa, 0x51, 0x6e, 0xd2, 0x75, 0x02, 0x07, 0x2a,
	0x68, 0xb9, 0xa2, 0x6a, 0xd7, 0xd8, 0x32, 0x4a, 0xa5, 0x5e, 0x68, 0x23, 0x7a, 0x1f, 0xfa, 0x5c,
	0xfe, 0x92, 0x6a, 0x09, 0x1e, 0x6d, 0xfd, 0xac, 0xc2, 0x3a, 0x02, 0x7d, 0xf9, 0x02, 0x8f, 0x7d,
	0x19, 0x3f, 0x69, 0xf3, 0x78, 0x17, 0x04, 0x76, 0xdd, 0xde, 0xa5, 0xb3, 0xd7, 0x73, 0xfb, 0x8f,
	0x83, 0x9f, 0x27, 0xa2, 0x9e, 0x0f, 0x54, 0x41, 0x31, 0x79, 0x36, 0xdd, 0x5c, 0xa7, 0xe5, 0x6d,
	0x32, 0x2d, 0xe7, 0xbf, 0x59, 0x83, 0x1f, 0x4a, 0x42, 0x65, 0xb1, 0xf3, 0x9e, 0x04, 0xfd, 0xf0,
	0x9f, 0x00, 0x00, 0x00, 0xff, 0xff, 0x7f, 0x53, 0xe9, 0x37, 0xbc, 0x08, 0x00, 0x00,
}
