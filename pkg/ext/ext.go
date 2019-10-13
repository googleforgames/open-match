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

// Package ext simplifies usage of repeated any fields used for extensions
// in Open Match protos.  Multiple values of the same type are not allowed.
package ext

import (
	"errors"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
)

var ErrNotFound = errors.New("Message type not found in extensions.")
var ErrMultipleFound = errors.New("Message type found multiple times in extensions.")

func Unmarshal(exts []*any.Any, pb proto.Message) error {
	var a *any.Any
	for _, ext := range exts {
		if ptypes.Is(ext, pb) {
			if a != nil {
				return ErrMultipleFound
			}
			a = ext
		}
	}

	return ptypes.UnmarshalAny(a, pb)
}

func MarshalMany(pbs ...proto.Message) ([]*any.Any, error) {
	result := make([]*any.Any, len(pbs))
	for i, pb := range pbs {
		var err error
		result[i], err = ptypes.MarshalAny(pb)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func MustMarshalMany(pbs ...proto.Message) []*any.Any {
	result, err := MarshalMany(pbs...)
	if err != nil {
		panic(err)
	}
	return result
}
