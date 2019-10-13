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

// ErrNotFound indicates that no any values of the given type were found.
var ErrNotFound = errors.New("Message type not found in extensions.")

// ErrMultipleFound indicates that multiple messages of a the given type were
// found.  Within Open Match, such extension fields are invalid.
var ErrMultipleFound = errors.New("Message type found multiple times in extensions.")

// Finds the any value which matches the passed message's type.
// Returns ErrNotFound if the given type isn't found.
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

	if a == nil {
		return ErrNotFound
	}

	return ptypes.UnmarshalAny(a, pb)
}

// MarshalMany takes any number of proto messages of different types
// and converts them into a slice of any values.  Returns ErrMultipleFound
// if any of the values are of the same type.
func MarshalMany(pbs ...proto.Message) ([]*any.Any, error) {
	result := make([]*any.Any, len(pbs))
	for i, pb := range pbs {
		var err error
		result[i], err = ptypes.MarshalAny(pb)
		if err != nil {
			return nil, err
		}
	}

	err := ValidateAnysAreUnique(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// MustMarshalMany is a convienance to call MarshalMany in testing.
// If an error occurs, MustMarshalMany panics.
func MustMarshalMany(pbs ...proto.Message) []*any.Any {
	result, err := MarshalMany(pbs...)
	if err != nil {
		panic(err)
	}
	return result
}

const validateAnysAreUniqueFastPath = 10

// ValidateAnysAreUnique returns an error if the type of two values is
// the same.
func ValidateAnysAreUnique(anys []*any.Any) error {
	if len(anys) <= validateAnysAreUniqueFastPath {
		for i, a := range anys {
			for _, b := range anys[i+1:] {
				if a.TypeUrl == b.TypeUrl {
					return ErrMultipleFound
				}
			}
		}
	} else {
		s := map[string]struct{}{}
		for _, a := range anys {
			_, ok := s[a.TypeUrl]
			if !ok {
				return ErrMultipleFound
			}
			s[a.TypeUrl] = struct{}{}
		}
	}
	return nil
}
