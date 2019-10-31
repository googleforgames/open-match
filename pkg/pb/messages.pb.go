// Code generated by protoc-gen-go. DO NOT EDIT.
// source: api/messages.proto

package pb

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	any "github.com/golang/protobuf/ptypes/any"
	status "google.golang.org/genproto/googleapis/rpc/status"
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

// A Ticket is a basic matchmaking entity in Open Match. A Ticket represents either an
// individual 'Player' or a 'Group' of players. Open Match will not interpret
// what the Ticket represents but just treat it as a matchmaking unit with a set
// of SearchFields. Open Match stores the Ticket in state storage and enables an
// Assignment to be associated with this Ticket.
type Ticket struct {
	// Id represents an auto-generated Id issued by Open Match.
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	// An Assignment represents a game server assignment associated with a Ticket.
	// Open Match does not require or inspect any fields on Assignment.
	Assignment *Assignment `protobuf:"bytes,3,opt,name=assignment,proto3" json:"assignment,omitempty"`
	// Search fields are the fields which Open Match is aware of, and can be used
	// when specifying filters.
	SearchFields *SearchFields `protobuf:"bytes,4,opt,name=search_fields,json=searchFields,proto3" json:"search_fields,omitempty"`
	// Customized information to be used by the Match Making Function.  Optional,
	// depending on the requirements of the MMF.
	Extension            *any.Any `protobuf:"bytes,5,opt,name=extension,proto3" json:"extension,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
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

func (m *Ticket) GetExtension() *any.Any {
	if m != nil {
		return m.Extension
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

// An Assignment represents a game server assignment associated with a Ticket. Open
// match does not require or inspect any fields on assignment.
type Assignment struct {
	// Connection information for this Assignment.
	Connection string `protobuf:"bytes,1,opt,name=connection,proto3" json:"connection,omitempty"`
	// Error when finding an Assignment for this Ticket.
	Error *status.Status `protobuf:"bytes,3,opt,name=error,proto3" json:"error,omitempty"`
	// Customized information to be sent to the clients.  Optional, depending on
	// what callers are expecting.
	Extension            *any.Any `protobuf:"bytes,4,opt,name=extension,proto3" json:"extension,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
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

func (m *Assignment) GetError() *status.Status {
	if m != nil {
		return m.Error
	}
	return nil
}

func (m *Assignment) GetExtension() *any.Any {
	if m != nil {
		return m.Extension
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
	// Maximum value. Defaults to positive infinity (any value above minv).
	Max float64 `protobuf:"fixed64,2,opt,name=max,proto3" json:"max,omitempty"`
	// Minimum value. Defaults to 0.
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

type Pool struct {
	// A developer-chosen human-readable name for this Pool.
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// Set of Filters indicating the filtering criteria. Selected players must
	// match every Filter.
	DoubleRangeFilters   []*DoubleRangeFilter  `protobuf:"bytes,2,rep,name=double_range_filters,json=doubleRangeFilters,proto3" json:"double_range_filters,omitempty"`
	StringEqualsFilters  []*StringEqualsFilter `protobuf:"bytes,4,rep,name=string_equals_filters,json=stringEqualsFilters,proto3" json:"string_equals_filters,omitempty"`
	TagPresentFilters    []*TagPresentFilter   `protobuf:"bytes,5,rep,name=tag_present_filters,json=tagPresentFilters,proto3" json:"tag_present_filters,omitempty"`
	XXX_NoUnkeyedLiteral struct{}              `json:"-"`
	XXX_unrecognized     []byte                `json:"-"`
	XXX_sizecache        int32                 `json:"-"`
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

// A Roster is a named collection of Ticket IDs. It exists so that a Tickets
// associated with a Match can be labelled to belong to a team, sub-team etc. It
// can also be used to represent the current state of a Match in scenarios such
// as backfill, join-in-progress etc.
type Roster struct {
	// A developer-chosen human-readable name for this Roster.
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// Tickets belonging to this Roster.
	TicketIds            []string `protobuf:"bytes,2,rep,name=ticket_ids,json=ticketIds,proto3" json:"ticket_ids,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Roster) Reset()         { *m = Roster{} }
func (m *Roster) String() string { return proto.CompactTextString(m) }
func (*Roster) ProtoMessage()    {}
func (*Roster) Descriptor() ([]byte, []int) {
	return fileDescriptor_cb9fb1f207fd5b8c, []int{7}
}

func (m *Roster) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Roster.Unmarshal(m, b)
}
func (m *Roster) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Roster.Marshal(b, m, deterministic)
}
func (m *Roster) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Roster.Merge(m, src)
}
func (m *Roster) XXX_Size() int {
	return xxx_messageInfo_Roster.Size(m)
}
func (m *Roster) XXX_DiscardUnknown() {
	xxx_messageInfo_Roster.DiscardUnknown(m)
}

var xxx_messageInfo_Roster proto.InternalMessageInfo

func (m *Roster) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Roster) GetTicketIds() []string {
	if m != nil {
		return m.TicketIds
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
	// The pool names can be used in empty Rosters to specify composition of a
	// match.
	Pools []*Pool `protobuf:"bytes,3,rep,name=pools,proto3" json:"pools,omitempty"`
	// Set of Rosters for this match request. Could be empty Rosters used to
	// indicate the composition of the generated Match or they could be partially
	// pre-populated Ticket list to be used in scenarios such as backfill / join
	// in progress.
	Rosters []*Roster `protobuf:"bytes,4,rep,name=rosters,proto3" json:"rosters,omitempty"`
	// Customized information on how the match function should run.  Optional,
	// depending on the requirements of the match function.
	Extension            *any.Any `protobuf:"bytes,5,opt,name=extension,proto3" json:"extension,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *MatchProfile) Reset()         { *m = MatchProfile{} }
func (m *MatchProfile) String() string { return proto.CompactTextString(m) }
func (*MatchProfile) ProtoMessage()    {}
func (*MatchProfile) Descriptor() ([]byte, []int) {
	return fileDescriptor_cb9fb1f207fd5b8c, []int{8}
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

func (m *MatchProfile) GetRosters() []*Roster {
	if m != nil {
		return m.Rosters
	}
	return nil
}

func (m *MatchProfile) GetExtension() *any.Any {
	if m != nil {
		return m.Extension
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
	// Set of Rosters that comprise this Match
	Rosters []*Roster `protobuf:"bytes,5,rep,name=rosters,proto3" json:"rosters,omitempty"`
	// Customized information for the evaluator.  Optional, depending on the
	// requirements of the configured evaluator.
	EvaluationInput *any.Any `protobuf:"bytes,7,opt,name=evaluation_input,json=evaluationInput,proto3" json:"evaluation_input,omitempty"`
	// Customized information for how the caller of FetchMatches should handle
	// this match.  Optional, depending on the requirements of the FetchMatches
	// caller.
	Extension            *any.Any `protobuf:"bytes,8,opt,name=extension,proto3" json:"extension,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Match) Reset()         { *m = Match{} }
func (m *Match) String() string { return proto.CompactTextString(m) }
func (*Match) ProtoMessage()    {}
func (*Match) Descriptor() ([]byte, []int) {
	return fileDescriptor_cb9fb1f207fd5b8c, []int{9}
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

func (m *Match) GetRosters() []*Roster {
	if m != nil {
		return m.Rosters
	}
	return nil
}

func (m *Match) GetEvaluationInput() *any.Any {
	if m != nil {
		return m.EvaluationInput
	}
	return nil
}

func (m *Match) GetExtension() *any.Any {
	if m != nil {
		return m.Extension
	}
	return nil
}

func init() {
	proto.RegisterType((*Ticket)(nil), "openmatch.Ticket")
	proto.RegisterType((*SearchFields)(nil), "openmatch.SearchFields")
	proto.RegisterMapType((map[string]float64)(nil), "openmatch.SearchFields.DoubleArgsEntry")
	proto.RegisterMapType((map[string]string)(nil), "openmatch.SearchFields.StringArgsEntry")
	proto.RegisterType((*Assignment)(nil), "openmatch.Assignment")
	proto.RegisterType((*DoubleRangeFilter)(nil), "openmatch.DoubleRangeFilter")
	proto.RegisterType((*StringEqualsFilter)(nil), "openmatch.StringEqualsFilter")
	proto.RegisterType((*TagPresentFilter)(nil), "openmatch.TagPresentFilter")
	proto.RegisterType((*Pool)(nil), "openmatch.Pool")
	proto.RegisterType((*Roster)(nil), "openmatch.Roster")
	proto.RegisterType((*MatchProfile)(nil), "openmatch.MatchProfile")
	proto.RegisterType((*Match)(nil), "openmatch.Match")
}

func init() { proto.RegisterFile("api/messages.proto", fileDescriptor_cb9fb1f207fd5b8c) }

var fileDescriptor_cb9fb1f207fd5b8c = []byte{
	// 777 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xa4, 0x54, 0xdd, 0x6e, 0xeb, 0x44,
	0x10, 0x96, 0x7f, 0xf2, 0xe3, 0x49, 0xce, 0x69, 0xba, 0xa7, 0x55, 0xdd, 0x42, 0x51, 0x64, 0xa8,
	0x88, 0x84, 0x70, 0xa4, 0x20, 0x24, 0xc4, 0x8f, 0x50, 0x11, 0xad, 0x48, 0x11, 0x50, 0xb6, 0x15,
	0x17, 0xdc, 0x44, 0x9b, 0x78, 0xe3, 0x5a, 0x75, 0xd6, 0x66, 0x77, 0x53, 0x35, 0x6f, 0xd1, 0xe7,
	0xe0, 0x8a, 0x0b, 0xde, 0x80, 0x0b, 0x5e, 0x0b, 0xed, 0xae, 0xed, 0xb8, 0x69, 0x68, 0x41, 0xe7,
	0x6e, 0x77, 0xe6, 0x9b, 0x6f, 0x67, 0xbe, 0x6f, 0x6c, 0x40, 0x24, 0x4f, 0x86, 0x0b, 0x2a, 0x04,
	0x89, 0xa9, 0x08, 0x73, 0x9e, 0xc9, 0x0c, 0x79, 0x59, 0x4e, 0xd9, 0x82, 0xc8, 0xd9, 0xcd, 0xd1,
	0x41, 0x9c, 0x65, 0x71, 0x4a, 0x87, 0x3c, 0x9f, 0x0d, 0x85, 0x24, 0x72, 0x59, 0x60, 0x8e, 0x0e,
	0x8b, 0x84, 0xbe, 0x4d, 0x97, 0xf3, 0x21, 0x61, 0x2b, 0x93, 0x0a, 0xfe, 0xb6, 0xa0, 0x79, 0x9d,
	0xcc, 0x6e, 0xa9, 0x44, 0xaf, 0xc1, 0x4e, 0x22, 0xdf, 0xea, 0x5b, 0x03, 0x0f, 0xdb, 0x49, 0x84,
	0x3e, 0x05, 0x20, 0x42, 0x24, 0x31, 0x5b, 0x50, 0x26, 0x7d, 0xa7, 0x6f, 0x0d, 0x3a, 0xa3, 0xfd,
	0xb0, 0x7a, 0x2e, 0x3c, 0xad, 0x92, 0xb8, 0x06, 0x44, 0x5f, 0xc2, 0x2b, 0x41, 0x09, 0x9f, 0xdd,
	0x4c, 0xe6, 0x09, 0x4d, 0x23, 0xe1, 0xbb, 0xba, 0xf2, 0xa0, 0x56, 0x79, 0xa5, 0xf3, 0xe7, 0x3a,
	0x8d, 0xbb, 0xa2, 0x76, 0x43, 0x23, 0xf0, 0xe8, 0xbd, 0xa4, 0x4c, 0x24, 0x19, 0xf3, 0x1b, 0xba,
	0x72, 0x2f, 0x34, 0xed, 0x87, 0x65, 0xfb, 0xe1, 0x29, 0x5b, 0xe1, 0x35, 0xec, 0xc2, 0x6d, 0xdb,
	0x3d, 0x27, 0xf8, 0xd3, 0x86, 0x6e, 0x9d, 0x18, 0x7d, 0x07, 0x9d, 0x28, 0x5b, 0x4e, 0x53, 0x3a,
	0x21, 0x3c, 0x16, 0xbe, 0xd5, 0x77, 0x06, 0x9d, 0xd1, 0x87, 0xff, 0xd2, 0x46, 0xf8, 0xad, 0x86,
	0x9e, 0xf2, 0x58, 0x9c, 0x31, 0xc9, 0x57, 0x18, 0xa2, 0x2a, 0xa0, 0x98, 0x84, 0xe4, 0x09, 0x8b,
	0x0d, 0x93, 0xfd, 0x3c, 0xd3, 0x95, 0x86, 0xd6, 0x98, 0x44, 0x15, 0x40, 0x08, 0x5c, 0x49, 0x62,
	0xe1, 0x3b, 0x7d, 0x67, 0xe0, 0x61, 0x7d, 0x3e, 0xfa, 0x0a, 0x76, 0x36, 0x1e, 0x47, 0x3d, 0x70,
	0x6e, 0xe9, 0xaa, 0xf0, 0x42, 0x1d, 0xd1, 0x1e, 0x34, 0xee, 0x48, 0xba, 0xa4, 0xbe, 0xdd, 0xb7,
	0x06, 0x16, 0x36, 0x97, 0xcf, 0xed, 0xcf, 0x2c, 0x55, 0xbe, 0xf1, 0xe2, 0x4b, 0xe5, 0x5e, 0xad,
	0x3c, 0x78, 0xb0, 0x00, 0xd6, 0x4e, 0xa2, 0xf7, 0x00, 0x66, 0x19, 0x63, 0x74, 0x26, 0x95, 0x01,
	0x86, 0xa1, 0x16, 0x41, 0x03, 0x68, 0x50, 0xce, 0x33, 0x5e, 0xec, 0x03, 0x2a, 0xbd, 0xe1, 0xf9,
	0x2c, 0xbc, 0xd2, 0x3b, 0x87, 0x0d, 0xe0, 0xb1, 0x93, 0xee, 0xff, 0x71, 0xf2, 0x17, 0xd8, 0x35,
	0x82, 0x60, 0xc2, 0x62, 0x7a, 0x9e, 0xa4, 0x92, 0x72, 0x74, 0x0c, 0xb0, 0x76, 0xb3, 0x68, 0xcc,
	0xab, 0x3c, 0x52, 0x23, 0x2f, 0xc8, 0x7d, 0xa1, 0x8e, 0x3a, 0xea, 0x48, 0xc2, 0x74, 0x9f, 0x2a,
	0x92, 0xb0, 0x60, 0x0c, 0xc8, 0x28, 0x75, 0xf6, 0xdb, 0x92, 0xa4, 0x62, 0x4d, 0xbc, 0x36, 0xb7,
	0x24, 0xae, 0x2c, 0xdb, 0xae, 0x5c, 0xf0, 0x01, 0xf4, 0xae, 0x49, 0x7c, 0xc9, 0xa9, 0xa0, 0x4c,
	0x16, 0x44, 0x3d, 0x70, 0x24, 0x29, 0x19, 0xd4, 0x31, 0x78, 0xb0, 0xc1, 0xbd, 0xcc, 0xb2, 0x54,
	0xd9, 0xce, 0xc8, 0x82, 0x16, 0x39, 0x7d, 0x46, 0x3f, 0xc2, 0x5e, 0x31, 0x10, 0x57, 0x63, 0x4e,
	0xe6, 0x9a, 0xa5, 0xdc, 0xae, 0x77, 0x6b, 0xdb, 0xf5, 0x44, 0x0c, 0x8c, 0xa2, 0xcd, 0x90, 0x40,
	0x3f, 0xc3, 0x7e, 0x31, 0x07, 0xd5, 0xe3, 0x55, 0x84, 0xae, 0x26, 0x3c, 0xae, 0xaf, 0xeb, 0x13,
	0x15, 0xf0, 0x1b, 0xf1, 0x24, 0x26, 0xd0, 0xf7, 0xf0, 0x46, 0x92, 0x78, 0x92, 0x9b, 0x31, 0x2b,
	0xc2, 0x86, 0x26, 0x7c, 0xa7, 0x46, 0xb8, 0xa9, 0x05, 0xde, 0x95, 0x1b, 0x11, 0x71, 0xe1, 0xb6,
	0x9d, 0x9e, 0x1b, 0x7c, 0x01, 0x4d, 0x9c, 0x09, 0x25, 0xd7, 0x36, 0x4d, 0x8e, 0x01, 0xa4, 0xfe,
	0x19, 0x4d, 0x92, 0xc8, 0x28, 0xe1, 0x61, 0xcf, 0x44, 0xc6, 0x91, 0x08, 0xfe, 0xb0, 0xa0, 0xfb,
	0x83, 0x7a, 0xf0, 0x92, 0x67, 0xf3, 0x24, 0xa5, 0x5b, 0x39, 0x4e, 0xa0, 0x91, 0x67, 0x59, 0x6a,
	0xbe, 0xb1, 0xce, 0x68, 0xa7, 0xd6, 0xa6, 0xf2, 0x02, 0x9b, 0x2c, 0xfa, 0x08, 0x5a, 0x5c, 0x37,
	0x52, 0x0a, 0xb4, 0x5b, 0x03, 0x9a, 0x16, 0x71, 0x89, 0x78, 0x8b, 0xbf, 0xd2, 0x5f, 0x36, 0x34,
	0x74, 0xcb, 0xe8, 0x10, 0xda, 0x9a, 0x7c, 0x52, 0xfd, 0x64, 0x5b, 0xfa, 0x3e, 0x8e, 0xd0, 0xfb,
	0xf0, 0xca, 0xa4, 0x72, 0x33, 0x57, 0xb1, 0x6b, 0xdd, 0x45, 0x7d, 0xd6, 0x13, 0x78, 0x6d, 0x40,
	0xf3, 0x25, 0x33, 0x5f, 0xa7, 0xa3, 0x51, 0xa6, 0xf4, 0xbc, 0x08, 0xaa, 0xb9, 0x8c, 0x60, 0xdb,
	0xe6, 0x32, 0x7f, 0x7a, 0x5c, 0x22, 0xea, 0x22, 0x34, 0x5e, 0x14, 0xe1, 0x6b, 0xe8, 0x51, 0xb5,
	0xfd, 0x44, 0xbd, 0x33, 0x49, 0x58, 0xbe, 0x94, 0x7e, 0xeb, 0x19, 0x2d, 0x76, 0xd6, 0xe8, 0xb1,
	0x02, 0x3f, 0x56, 0xb1, 0xfd, 0x5f, 0x55, 0x6c, 0xf6, 0x5a, 0xdf, 0x84, 0xbf, 0xf6, 0x55, 0x5f,
	0x1f, 0x9b, 0xc6, 0x22, 0x7a, 0x37, 0x5c, 0x5f, 0x87, 0xf9, 0x6d, 0x3c, 0xcc, 0xa7, 0xbf, 0xdb,
	0xde, 0x4f, 0x39, 0x65, 0x5a, 0xeb, 0x69, 0x53, 0xd3, 0x7d, 0xf2, 0x4f, 0x00, 0x00, 0x00, 0xff,
	0xff, 0xed, 0x2a, 0x50, 0x0d, 0x31, 0x07, 0x00, 0x00,
}
