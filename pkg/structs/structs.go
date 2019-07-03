// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package structs simplifies the specification of protobuf's struct literals.
// This package has nothing to do with Go's structs, or protobufs in general.
package structs

import (
	structpb "github.com/golang/protobuf/ptypes/struct"
)

// Bool converts a boolean value into a proto struct value.
func Bool(v bool) *structpb.Value {
	return &structpb.Value{
		Kind: &structpb.Value_BoolValue{
			BoolValue: v,
		},
	}
}

// List is used to specify a proto list.  It can be used either as a literal, or
// like you would a normal slice.  Call L() or V() when finished constructing to
// retrieve the type you want.
type List []*structpb.Value

// L converts a List into a proto struct list.
func (l List) L() *structpb.ListValue {
	return &structpb.ListValue{
		Values: ([]*structpb.Value)(l),
	}
}

// V converts a List into a proto struct value.
func (l List) V() *structpb.Value {
	return &structpb.Value{
		Kind: &structpb.Value_ListValue{
			ListValue: l.L(),
		},
	}
}

// Null returns a proto struct value's null.
func Null() *structpb.Value {
	return &structpb.Value{
		Kind: &structpb.Value_NullValue{
			NullValue: structpb.NullValue_NULL_VALUE,
		},
	}
}

// Number converts a float64 into a proto struct value.
func Number(v float64) *structpb.Value {
	return &structpb.Value{
		Kind: &structpb.Value_NumberValue{
			NumberValue: v,
		},
	}
}

// String converts a string into a proto struct value.
func String(v string) *structpb.Value {
	return &structpb.Value{
		Kind: &structpb.Value_StringValue{
			StringValue: v,
		},
	}
}

// Struct is used to specify a proto struct.  It can be used either as a literal,
// or like you would a normal map.  Call S() or V() when finished constructing
// to retrieve the type you want.
type Struct map[string]*structpb.Value

// S converts a Struct into a proto struct struct.
func (s Struct) S() *structpb.Struct {
	return &structpb.Struct{
		Fields: (map[string]*structpb.Value)(s),
	}
}

// V converts a Struct into a proto struct value.
func (s Struct) V() *structpb.Value {
	return &structpb.Value{
		Kind: &structpb.Value_StructValue{
			StructValue: s.S(),
		},
	}
}
